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
	localRoot             string           // where to store dl'd files
	allTypes              map[string]TType // all the types
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
		allTypes:  map[string]TType{},
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
	// fmt.Printf("%# v\n\n", pretty.Formatter(g.wsdl.Binding))
	// fmt.Printf("%# v\n\n", pretty.Formatter(g.wsdl.PortTypes))
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
	funcMap := template.FuncMap{
		"toGoType":             toGoType,
		"stripns":              stripns,
		"replaceReservedWords": replaceReservedWords,
		"makePublic":           makePublic,
		"comment":              comment,
	}

	g.flattenTypes()

	fmt.Printf("%# v\n\n", pretty.Formatter(g.allTypes))

	// data := new(bytes.Buffer)
	// tmpl := template.Must(template.New("types").Funcs(funcMap).Parse(typesTmpl))
	// err := tmpl.Execute(data, g.wsdl.Types)
	// if err != nil {
	// 	return nil, err
	// }

	data := new(bytes.Buffer)
	tmpl := template.Must(template.New("types").Funcs(funcMap).Parse(flatTypesTmpl))
	err := tmpl.Execute(data, g.allTypes)
	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

type TRestriction interface{}

type SimpleType struct {
	Comment      string
	Name         string
	Type         string
	Restrictions []TRestriction // optional
}

func (s SimpleType) TypeString() string {
	if len(s.Restrictions) > 0 { // if we have a restricted type then return this type
		return s.Name
	}
	return s.Type // else derefernce simple type
}

func (s SimpleType) FieldsRef() []*Field {
	return nil
}

func (s SimpleType) isComplex() bool {
	return false
}

func (s SimpleType) String() string {
	return "type\t" + s.Name + "\t" + toGoType(s.Type) + "\n"
}

func (s SimpleType) CommentString() string {
	return s.Comment
}

// Does not appear to serve any purpose other than a lookup
type Lookup struct {
	Comment string
	Name    string
	Type    string
}

func (l Lookup) TypeString() string {
	return l.Type
}

func (l Lookup) FieldsRef() []*Field {
	return nil
}

func (l Lookup) isComplex() bool {
	return false
}

func (l Lookup) String() string {
	return ""
}

func (l Lookup) CommentString() string {
	return l.Comment
}

type Field struct {
	Comment  string
	Name     string
	Type     string
	XMLTag   string
	Optional bool
	RefName  string
}

func makeComment(c string) (string, bool) {
	c = comment(strings.Trim(c, "\n\t"))

	nli := strings.Index(c, "\n")
	return c, nli >= 0
}

func (f *Field) String() string {
	comment := ""
	var multi bool
	if f.Comment != "" {
		comment, multi = makeComment(f.Comment)
	}
	if f.Name == "" {
		return ""
	}
	if multi {
		return fmt.Sprintf("%s\n\t%s\t%s\t%s", comment, makePublic(f.Name), toGoType(f.Type), f.XMLTag)
	}

	return fmt.Sprintf("%s\t%s\t%s\t%s", makePublic(f.Name), toGoType(f.Type), f.XMLTag, comment)

}

type Struct struct {
	Raw       string
	Comment   string
	Name      string
	NameSpace string
	XMLTag    string
	Fields    []*Field
}

// TODO
/*
{{if ne .ComplexContent.Extension.Base ""}}
			{{template "ComplexContent" .ComplexContent}}
		{{else if ne .SimpleContent.Extension.Base ""}}
			{{template "SimpleContent" .SimpleContent}}
*/

func (s Struct) String() string {

	s.Comment, _ = makeComment(s.Comment)

	tmpl := `
	{{.Comment}}
	type {{.Name}} struct {
		XMLName xml.Name 
		
		{{range .FieldsRef}}{{.String}}
		{{end}} }
`

	structTmpl, err := template.New("test").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	buf := &bytes.Buffer{}
	err = structTmpl.Execute(buf, s)
	if err != nil {
		panic(err)
	}

	return buf.String()
}

func (s Struct) TypeString() string {
	return s.Name
}

func (s Struct) FieldsRef() []*Field {
	return s.Fields
}

func (s Struct) isComplex() bool {
	return true
}

func (s Struct) CommentString() string {
	return s.Comment
}

type TType interface {
	String() string
	TypeString() string
	CommentString() string
	FieldsRef() []*Field
	isComplex() bool // shoudl we output this
}

// and returns a type name if it was a type
func (g *GoWSDL) flattenElement(s *XSDSchema, ct *XSDElement) string {
	if ct == nil {
		return ""
	}

	// ref elements will have been added
	if ct.Ref != "" {
		return "" // refs found later
	}

	// if this has not got a name then we cant make a type from it
	if ct.Name == "" {
		return ""
	}

	// when it has a name and a type I think it is only ever used as a lookup - so just keep it in
	// or other routines to use but skip it - TODO - check this - maybe add lookup to get the other details
	if ct.Type != "" {
		// add a lookup
		t, ok := g.allTypes[ct.Name]
		if ok {
			return getTypeName(t.TypeString()) // so just return the type
		}

		l := Lookup{
			Comment: ct.Doc,
			Name:    ct.Name,
			Type:    getTypeName(ct.Type),
		}

		g.allTypes[ct.Name] = l
		return getTypeName(ct.Type) // so just return the type
	}

	if ct.ComplexType != nil && ct.SimpleType != nil {
		panic("cant have simple and complex types in an element")
	}

	// if we have a name and no type then this is likely to be used as a type from a ref
	typeName := getTypeName(ct.Name)

	if ct.SimpleType != nil {
		// inject our name
		if ct.SimpleType.Name == "" {
			ct.SimpleType.Name = ct.Name
		}
		typeName = g.flattenSimpleType(s, ct.SimpleType)
	}

	if ct.ComplexType != nil {

		log.Println("Got complex element: ", ct.Name)

		// inject our name
		if ct.ComplexType.Name == "" {
			ct.ComplexType.Name = ct.Name
		}

		typeName = g.flattenComplexType(s, ct.ComplexType)

		log.Println("complex element created a type : ", typeName)
	}

	return typeName
}

func getTypeName(in string) string {
	return makePublic(replaceReservedWords(stripns(in)))
}

func (g *GoWSDL) flattenSimpleType(s *XSDSchema, t *XSDSimpleType) string {
	if t == nil {
		return ""
	}

	if t.Restriction == nil {
		log.Println("WARNING - simple type without a restriction - whats the point? " + t.Name)
	}

	typeName := getTypeName(t.Name)
	_, ok := g.allTypes[typeName]
	if ok {
		log.Println("WARNING - fst type name conflict " + typeName)
		return ""
	}

	sst := SimpleType{
		Name: typeName,
		Type: t.Restriction.Base,
		// Restrictions []TRestriction // TODO optional
	}

	g.allTypes[typeName] = sst

	return typeName
}

func (g *GoWSDL) flattenComplexType(s *XSDSchema, t *XSDComplexType) string {
	if t == nil {
		return ""
	}

	if t.Abstract {
		log.Println("abstract types not supported - skipping")
		return ""
	}

	typeName := getTypeName(t.Name)
	_, ok := g.allTypes[typeName]
	if ok {
		log.Println("WARNING - fct type name conflict " + typeName)
		return ""
	}

	sct := Struct{
		Raw:     t.Name,
		Name:    typeName,
		XMLTag:  fmt.Sprintf("`xml:\"%s\"`", s.TargetNamespace),
		Comment: t.Doc,
		Fields:  []*Field{},
	}

	for _, ct := range t.Attributes {
		ft := g.flattenSimpleType(s, ct.SimpleType)
		if ft == "" {
			ft = ct.Type
		}
		sct.Fields = append(sct.Fields, &Field{
			Comment:  ct.Doc,
			Name:     ct.Name,
			Type:     ft,
			XMLTag:   fmt.Sprintf("`xml:\"%s,attr,omitempty\"`", ct.Name),
			Optional: false,
			RefName:  "",
		})
	}

	for _, ct := range t.All {
		ft := g.flattenElement(s, ct)
		sct.Fields = append(sct.Fields, g.makeField(s, ct, ft))
	}
	for _, ct := range t.Choice {
		ft := g.flattenElement(s, ct)
		sct.Fields = append(sct.Fields, g.makeField(s, ct, ft))
	}
	for _, ct := range t.Sequence {
		ft := g.flattenElement(s, ct)
		sct.Fields = append(sct.Fields, g.makeField(s, ct, ft))
	}
	for _, ct := range t.SequenceChoice {
		ft := g.flattenElement(s, ct)
		sct.Fields = append(sct.Fields, g.makeField(s, ct, ft))
	}

	g.allTypes[typeName] = sct

	return typeName
}

func (g *GoWSDL) makeField(s *XSDSchema, ct *XSDElement, fType string) *Field {
	oe := ",omitempty"
	if ct.MinOccurs != "0" {
		oe = ""
	}

	name := ct.Name
	if name == "" {
		name = ct.Ref // use a ref if name not set
	}
	tag := fmt.Sprintf("`xml:\"%s%s\"`", name, oe)

	return &Field{
		Comment:  ct.Doc,
		Name:     name,
		Type:     fType,
		XMLTag:   tag,
		Optional: false,
		RefName:  ct.Ref,
	}
}

func (g *GoWSDL) toGoType(xsdType string) string {
	// Handles name space, ie. xsd:string, xs:string
	r := strings.Split(xsdType, ":")

	t := r[0]

	if len(r) == 2 {
		t = r[1]
	}

	value := xsd2GoTypes[strings.ToLower(t)]

	if value != "" {
		return value
	}

	return "*" + replaceReservedWords(makePublic(t))
}

func (g *GoWSDL) flattenTypes() {
	for _, s := range g.wsdl.Types.Schemas {

		for _, ct := range s.Elements {
			g.flattenElement(s, ct)
		}

		for _, ct := range s.ComplexTypes {
			g.flattenComplexType(s, ct)
		}

		for _, ct := range s.SimpleType {
			g.flattenSimpleType(s, ct)
		}
	}

	// no go back round and fill in refs

	for _, t := range g.allTypes {
		fl := t.FieldsRef()
		if fl != nil {
			for _, f := range fl {
				if f.RefName == "" {
					continue
				}
				var ok bool
				l, ok := g.allTypes[f.RefName]
				if ok {
					f.Type = l.TypeString()
				} else {
					l, ok = g.allTypes[getTypeName(f.RefName)]
					if ok {
						f.Type = l.TypeString()
					}
				}
				if ok {
					if f.Comment == "" && l.CommentString() != "" {
						f.Comment = l.CommentString()
					}
				}
			}
		}
	}
}

func (g *GoWSDL) genOperations() ([]byte, error) {
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

var xsd2GoTypes = map[string]string{
	"string":        "string",
	"token":         "string",
	"float":         "float32",
	"double":        "float64",
	"decimal":       "float64",
	"integer":       "int32",
	"int":           "int32",
	"short":         "int16",
	"byte":          "int8",
	"long":          "int64",
	"boolean":       "bool",
	"datetime":      "time.Time",
	"date":          "time.Time",
	"time":          "time.Time",
	"base64binary":  "[]byte",
	"hexbinary":     "[]byte",
	"unsignedint":   "uint32",
	"unsignedshort": "uint16",
	"unsignedbyte":  "byte",
	"unsignedlong":  "uint64",
	"anytype":       "interface{}",
}

func toGoType(xsdType string) string {
	// Handles name space, ie. xsd:string, xs:string
	r := strings.Split(xsdType, ":")

	t := r[0]

	if len(r) == 2 {
		t = r[1]
	}

	value := xsd2GoTypes[strings.ToLower(t)]

	if value != "" {
		return value
	}

	return "*" + replaceReservedWords(makePublic(t))
}

type NameType struct {
	Name string
	Type string
}

func (n *NameType) FixupForTemplate() {
	n.Name = stripns(n.Name)
	n.Type = makePublic(replaceReservedWords(stripns(n.Type)))
}

// Given a message, finds its type.
//
// I'm not very proud of this function but
// it works for now and performance doesn't
// seem critical at this point
func (g *GoWSDL) findTypeFromMessage(message string) NameType {
	nt := g.findTypeFromMessageRaw(message)
	nt.FixupForTemplate()
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
							Name: el.Name,
							Type: el.Type,
						}
					}
					// else ... what was the point of the reference :/ XSD ftw
					return NameType{
						Name: el.Name,
						Type: el.Name,
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
