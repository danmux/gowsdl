// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gowsdl

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"
	"unicode"

	"github.com/kr/pretty"
)

const maxRecursion uint8 = 20

// GoWSDL defines the struct for WSDL generator.
type GoWSDL struct {
	file, pkg             string
	ignoreTLS             bool
	wsdl                  *WSDL
	resolvedXSDExternals  map[string]bool
	currentRecursionLevel uint8
	localRoot             string // where to store dl'd files
}

var cacheDir = filepath.Join(os.TempDir(), "gowsdl-cache")

func init() {
	log.Println("Create cache directory", cacheDir)
	err := os.MkdirAll(cacheDir, 0700)
	if err != nil {
		log.Println("Create cache directory", "error", err)
		os.Exit(1)
	}
}

var timeout = time.Duration(30 * time.Second)

func dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

func downloadFile(furl string, ignoreTLS bool, cRoot string) ([]byte, error) {
	parsedURL, err := url.Parse(furl)

	// explicit file scheme ?
	if parsedURL.Scheme == "file" {
		return ioutil.ReadFile(parsedURL.RequestURI())
	} else if cRoot != "" { // check cache on local file system
		path, name := filepath.Split(parsedURL.RequestURI())
		fname := filepath.Join(cRoot+path, name)

		log.Println("attempting to load from cache: ", fname)
		b, err := ioutil.ReadFile(fname)
		if err == nil {
			log.Println("file loaded from cache as: ", fname)
			return b, nil
		} else if os.IsNotExist(err) {
			log.Println("not in cache: fname")
		} else {
			log.Println("cache load failed: ", err)
		}
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: ignoreTLS,
		},
		Dial: dialTimeout,
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Get(furl)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// cache on local file system
	if cRoot != "" {
		path, name := filepath.Split(parsedURL.RequestURI())
		fp := cRoot + path
		err := os.MkdirAll(fp, 0777)
		if err != nil {
			return nil, err
		}
		fname := filepath.Join(fp, name)

		err = ioutil.WriteFile(fname, data, 0644)
		if err != nil {
			return nil, err
		}
		log.Println("file saved in cache as: ", fname)
	}

	return data, nil
}

// NewGoWSDL initializes WSDL generator.
func NewGoWSDL(file, pkg string, ignoreTLS bool, localRoot string) (*GoWSDL, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return nil, errors.New("WSDL file is required to generate Go proxy")
	}

	pkg = strings.TrimSpace(pkg)
	if pkg == "" {
		pkg = "myservice"
	}

	if localRoot != "" && len(localRoot) > 1 {
		log.Println("Setting file dl root folder:", localRoot)

		if localRoot[:2] == "~/" {
			usr, _ := user.Current()
			hd := usr.HomeDir

			localRoot = strings.Replace(localRoot, "~", hd, 1)
		}
	}

	return &GoWSDL{
		file:      file,
		pkg:       pkg,
		ignoreTLS: ignoreTLS,
		localRoot: localRoot,
	}, nil
}

// Start initiaties the code generation process by starting two goroutines: one
// to generate types and another one to generate operations.
func (g *GoWSDL) Start() (map[string][]byte, error) {
	gocode := make(map[string][]byte)

	err := g.unmarshal()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error

		gocode["types"], err = g.genTypes()
		if err != nil {
			log.Println("genTypes", "error", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error

		gocode["operations"], err = g.genOperations()
		if err != nil {
			log.Println(err)
		}
	}()

	wg.Wait()

	gocode["header"], err = g.genHeader()
	if err != nil {
		log.Println(err)
	}

	gocode["soap"], err = g.genSOAPClient()
	if err != nil {
		log.Println(err)
	}

	return gocode, nil
}

func (g *GoWSDL) unmarshal() error {
	var data []byte

	parsedURL, err := url.Parse(g.file)
	if parsedURL.Scheme == "" {
		log.Println("Reading", "file", g.file)

		data, err = ioutil.ReadFile(g.file)
		if err != nil {
			return err
		}
	} else {
		log.Println("Downloading", "file", g.file)

		data, err = downloadFile(g.file, g.ignoreTLS, g.localRoot)
		if err != nil {
			return err
		}
	}

	g.wsdl = new(WSDL)
	err = xml.Unmarshal(data, g.wsdl)
	if err != nil {
		return err
	}

	for _, schema := range g.wsdl.Types.Schemas {
		err = g.resolveXSDExternals(schema, parsedURL)
		if err != nil {
			return err
		}
	}

	// fmt.Printf("%# v\n\n", pretty.Formatter(g.wsdl.Types))
	// fmt.Printf("%# v\n\n", pretty.Formatter(g.wsdl.Service))
	fmt.Printf("%# v\n\n", pretty.Formatter(g.wsdl.Binding))
	fmt.Printf("%# v\n\n", pretty.Formatter(g.wsdl.PortTypes))
	// fmt.Printf("%# v\n\n", pretty.Formatter(g.wsdl.Messages))

	return nil
}

// add in any schemas that have any includes
func (g *GoWSDL) resolveXSDExternals(schema *XSDSchema, url *url.URL) error {
	for _, incl := range schema.Includes {
		log.Println("resolving: ", incl.SchemaLocation)

		location, err := url.Parse(incl.SchemaLocation)
		if err != nil {
			return err
		}

		_, schemaName := filepath.Split(location.Path)
		log.Println("resolving schema name: ", schemaName)
		if g.resolvedXSDExternals[schemaName] { // memoisation
			continue
		}

		schemaLocation := location.String()
		if !location.IsAbs() {
			if !url.IsAbs() {
				return fmt.Errorf("Unable to resolve external schema %s through WSDL URL %s", schemaLocation, url)
			}
			schemaLocation = url.Scheme + "://" + url.Host + schemaLocation
		}

		log.Println("Downloading external schema", "location", schemaLocation)

		data, err := downloadFile(schemaLocation, g.ignoreTLS, g.localRoot)
		newschema := new(XSDSchema)

		err = xml.Unmarshal(data, newschema)
		if err != nil {
			log.Println(err)
			return err
		}

		if len(newschema.Includes) > 0 &&
			maxRecursion > g.currentRecursionLevel {

			g.currentRecursionLevel++

			log.Printf("Entering recursion %d\n", g.currentRecursionLevel)
			g.resolveXSDExternals(newschema, url)
		}

		g.wsdl.Types.Schemas = append(g.wsdl.Types.Schemas, newschema)

		if g.resolvedXSDExternals == nil {
			g.resolvedXSDExternals = make(map[string]bool, maxRecursion)
		}
		g.resolvedXSDExternals[schemaName] = true
	}

	return nil
}

func (g *GoWSDL) genTypes() ([]byte, error) {
	// funcMap := template.FuncMap{
	// 	"toGoType":             toGoType,
	// 	"stripns":              stripns,
	// 	"replaceReservedWords": replaceReservedWords,
	// 	"makePublic":           makePublic,
	// 	"comment":              comment,
	// }

	// data := new(bytes.Buffer)
	// tmpl := template.Must(template.New("types").Funcs(funcMap).Parse(typesTmpl))
	// err := tmpl.Execute(data, g.wsdl.Types)
	// if err != nil {
	// 	return nil, err
	// }

	// data := new(bytes.Buffer)
	// tmpl := template.Must(template.New("types").Funcs(funcMap).Parse(flatTypesTmpl))
	// err := tmpl.Execute(data, g.allTypes)
	// if err != nil {
	// return nil, err
	// }

	// return data.Bytes(), nil

	return nil, nil
}

func (g *GoWSDL) genOperations() ([]byte, error) {

	// filter out none soap bindings
	// find all soap http bindings - the only binding we support
	for _, binding := range g.wsdl.Binding {
		if binding.SOAPBinding.Transport == "http://schemas.xmlsoap.org/soap/http" {

		} else {
			log.Println("found a binding we dont support", binding.Name, binding.Type)
		}
	}

	// for _, port := range g.wsdl.PortTypes {
	// 	binding.Type
	// }

	funcMap := template.FuncMap{
		"toGoType":             toGoType,
		"stripns":              stripns,
		"replaceReservedWords": replaceReservedWords,
		"makePublic":           makePublic,
		"findTypeFromMessage":  g.findTypeFromMessage,
		"findSOAPAction":       g.findSOAPAction,
		"findServiceAddress":   g.findServiceAddress,
	}

	data := new(bytes.Buffer)
	tmpl := template.Must(template.New("operations").Funcs(funcMap).Parse(opsTmpl))
	err := tmpl.Execute(data, g.wsdl.PortTypes)
	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

func (g *GoWSDL) genHeader() ([]byte, error) {
	funcMap := template.FuncMap{
		"toGoType":             toGoType,
		"stripns":              stripns,
		"replaceReservedWords": replaceReservedWords,
		"makePublic":           makePublic,
		"comment":              comment,
	}

	data := new(bytes.Buffer)
	tmpl := template.Must(template.New("header").Funcs(funcMap).Parse(headerTmpl))
	err := tmpl.Execute(data, g.pkg)
	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

func (g *GoWSDL) genSOAPClient() ([]byte, error) {
	data := new(bytes.Buffer)
	tmpl := template.Must(template.New("soapclient").Parse(soapTmpl))
	err := tmpl.Execute(data, g.pkg)
	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

var reservedWords = map[string]string{
	"break":       "break_",
	"default":     "default_",
	"func":        "func_",
	"interface":   "interface_",
	"select":      "select_",
	"case":        "case_",
	"defer":       "defer_",
	"go":          "go_",
	"map":         "map_",
	"struct":      "struct_",
	"chan":        "chan_",
	"else":        "else_",
	"goto":        "goto_",
	"package":     "package_",
	"switch":      "switch_",
	"const":       "const_",
	"fallthrough": "fallthrough_",
	"if":          "if_",
	"range":       "range_",
	"type":        "type_",
	"continue":    "continue_",
	"for":         "for_",
	"import":      "import_",
	"return":      "return_",
	"var":         "var_",
}

// Replaces Go reserved keywords to avoid compilation issues
func replaceReservedWords(identifier string) string {
	value := reservedWords[identifier]
	if value != "" {
		return value
	}
	return normalize(identifier)
}

// Normalizes value to be used as a valid Go identifier, avoiding compilation issues
func normalize(value string) string {
	mapping := func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}

	return strings.Map(mapping, value)
}

type NameType struct {
	Name      string
	Type      string
	Namespace string
}

func (n *NameType) FixupForTemplate() {
	n.Name = stripns(n.Name)
	// n.Type = toGoType(getTypeName(n.Type))
	if n.Type[0] == '*' { // strip pointers - as teplate addds them back
		n.Type = n.Type[1:]
	}
}

// Given a message, finds its type.
//
// I'm not very proud of this function but
// it works for now and performance doesn't
// seem critical at this point
func (g *GoWSDL) findTypeFromMessage(message string) NameType {
	nt := g.findTypeFromMessageRaw(message)
	println("==== = = = = = = ", nt.Type)
	nt.FixupForTemplate()

	println("==== = = = = = = ?", nt.Type)
	return nt
}

func (g *GoWSDL) findTypeFromMessageRaw(message string) NameType {
	message = stripns(message)

	for _, msg := range g.wsdl.Messages {
		if msg.Name != message {
			continue
		}

		// Assumes document/literal wrapped WS-I
		if len(msg.Parts) == 0 {
			// Message does not have parts. This could be a Port
			// with HTTP binding or SOAP 1.2 binding, which are not currently
			// supported.
			log.Printf("[WARN] %s message doesn't have any parts, ignoring message...", msg.Name)
			continue
		}

		// So SOAP 1 only supports one request param and one response
		part := msg.Parts[0]
		if part.Type != "" {
			return NameType{
				Name: part.Type,
				Type: part.Type,
			}
		}

		// otherwise look up the element reference
		elRef := stripns(part.Element) // TODO - support ns in case of naming conflicts

		for _, schema := range g.wsdl.Types.Schemas {
			for _, el := range schema.Elements {
				if strings.EqualFold(elRef, el.Name) {
					if el.Type != "" {
						return NameType{ // this needs to be used to create the xml element name as well
							Name:      el.Name,
							Type:      el.Type,
							Namespace: schema.TargetNamespace,
						}
					}
					// else ... what was the point of the reference :/ XSD ftw
					return NameType{
						Name:      el.Name,
						Type:      el.Name,
						Namespace: schema.TargetNamespace,
					}
				}
			}
		}
	}
	return NameType{
		Name: "",
		Type: "",
	}
}

// findSOAPAction - is used as a template function for genOperations
// portType must be the raw value not the public-ised one
// TODO(c4milo): Add support for namespaces instead of striping them out
// TODO(c4milo): improve runtime complexity if performance turns out to be an issue.
func (g *GoWSDL) findSOAPAction(operation, portType string) string {
	for _, binding := range g.wsdl.Binding {
		if stripns(binding.Type) != portType {
			continue
		}

		for _, soapOp := range binding.Operations {
			if soapOp.Name == operation {
				return soapOp.SOAPOperation.SOAPAction
			}
		}
	}
	return ""
}

func (g *GoWSDL) findServiceAddress(name string) string {
	for _, service := range g.wsdl.Service {
		for _, port := range service.Ports {
			if port.Name == name {
				return port.SOAPAddress.Location
			}
		}
	}
	return ""
}

// TODO(c4milo): Add namespace support instead of stripping it
func stripns(xsdType string) string {
	r := strings.Split(xsdType, ":")
	t := r[0]

	if len(r) == 2 {
		t = r[1]
	}

	return t
}

// make first character uppercase
func makePublic(identifier string) string {
	field := []rune(identifier)
	if len(field) == 0 {
		// return identifier
		panic("Can't make an empty id public")
	}

	field[0] = unicode.ToUpper(field[0])
	return string(field)
}

func comment(text string) string {
	lines := strings.Split(text, "\n")

	if len(lines) == 1 && lines[0] == "" {
		return ""
	}

	// Helps to determine if there is an actual comment without screwing newlines
	// in real comments.
	hasComment := false

	for _, line := range lines {
		line = strings.TrimLeftFunc(line, unicode.IsSpace)
		if line != "" {
			hasComment = true
		}
	}

	if hasComment {
		return " // " + strings.Join(lines, "\n// ")
	}
	return ""
}
