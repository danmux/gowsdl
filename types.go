// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gowsdl

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"text/template"

	"github.com/kr/pretty"
)

type schema struct {
	schema     *XSDSchema        // the schemas used to build these types
	schemaMeta *XSDSchemaMeta    // the captured schema metas - will be in same order as above
	elems      map[string]Field  // any elements at the top level that can be used via refs
	types      map[string]GoType // the types
	tnsPrefix  string            // target name space
	xsPrefix   string            // the prefix used for xsd http://www.w3.org/2001/XMLSchema
}

// types only support one namespace + xsd namespace
type types struct {
	schemas []*schema // schemas
}

func (ty *types) genTypes() ([]byte, error) {

	fmt.Printf("SCHEMAS:\n%# v\n\n", pretty.Formatter(ty.schemas))

	ty.buildTypes()

	fmt.Printf("ELEMS:\n%# v\n\n", pretty.Formatter(ty.elems))
	fmt.Printf("TYPES:\n%# v\n\n", pretty.Formatter(ty.types))

	data := new(bytes.Buffer)
	tmpl := template.Must(template.New("types").Parse(flatTypesTmpl))
	err := tmpl.Execute(data, sc.types)
	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

func (sc *schema) buildTypes() {
	sc.elems = map[string]Field{}
	sc.types = map[string]GoType{}

	for _, s := range sc.schemas {

		// xsd prefix
		// target prefix

		for _, ct := range s.Elements { // named fields and params
			sc.buildElement(s, ct, true)
		}

		for _, ct := range s.ComplexTypes {
			sc.buildComplexType(s, ct)
		}

		for _, ct := range s.SimpleType {
			sc.buildSimpleType(s, ct)
		}
	}

	// now go back round and fill in refs and finalising types

	// TODO - use the namespace prefixes
	for _, t := range sc.types {
		fl := t.FieldsRef()
		if fl != nil {
			for _, f := range fl {

				ttc := f.Type

				if f.RefName != "" {
					ttc = f.RefName
				}

				var ok bool
				l, ok := sc.types[ttc]
				if ok {
					f.Type = l.TypeString()
				}
				if ok {
					if f.Comment == "" && l.CommentString() != "" {
						f.Comment = l.CommentString()
					}
				}

				// f.Type = sc.toGoType(f.Type)
			}
		}
	}
}

// GoType encapsulates any type that can be rendered out as a golang type - base or struct
// implemented by SimpleType or Struct
type GoType interface {
	String() string
	TypeString() string
	CommentString() string
	FieldsRef() []*Field
	isComplex() bool // shoudl we output this
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

// a field can be a field or param to a function
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
		return fmt.Sprintf("%s\n\t%s\t%s\t%s", comment, makePublic(f.Name), f.Type, f.XMLTag)
	}

	return fmt.Sprintf("%s\t%s\t%s\t%s", makePublic(f.Name), f.Type, f.XMLTag, comment)

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

func (sc *schema) addBaseElement(name, eType, comment string) {
	if eType == "" { // cant add an element if it has no type
		return
	}

	e, ok := sc.elems[name]
	if ok {
		return
	}

	// add a lookup
	e = Field{
		Name:    name,
		Type:    eType,
		Comment: comment,
		XMLTag:  name,
	}

	sc.elems[name] = e
}

// returns a type name if it was a type base means its a top level element
func (sc *schema) buildElement(s *XSDSchema, ct *XSDElement, base bool) string {
	if ct == nil {
		return ""
	}

	// ref elements will have been added
	if ct.Ref != "" {
		return "" // refs found later
	}

	// if this has not got a name then we cant make an element from it
	if ct.Name == "" {
		log.Println("ERROR - element without a name")
		return ""
	}

	// ---- when it has a named type - just return the type, if a base element add it to the list of elems to look up
	if ct.Type != "" {
		if ct.ComplexType != nil || ct.SimpleType != nil {
			panic("found a base element with a type name and type definitions")
		}
		if base {
			sc.addBaseElement(ct.Name, ct.Type, ct.Doc)
		}
		return ct.Type
	}

	if ct.ComplexType != nil && ct.SimpleType != nil {
		panic("cant have both simple and complex types in an element")
	}

	if ct.ComplexType == nil && ct.SimpleType == nil {
		panic("cant have an element with no type no ref and no simple or complex types specified")
	}

	var typeName string

	// got a simple type
	if ct.SimpleType != nil {
		// inject our element name as this types name
		if ct.SimpleType.Name == "" {
			ct.SimpleType.Name = ct.Name
		}
		typeName = sc.buildSimpleType(s, ct.SimpleType)
		if base {
			sc.addBaseElement(ct.SimpleType.Name, typeName, ct.Doc)
		}
	}

	if ct.ComplexType != nil {
		log.Println("Got complex element type: ", ct.Name)

		// inject our element name as this types name
		if ct.ComplexType.Name == "" {
			ct.ComplexType.Name = ct.Name
		}

		typeName = sc.buildComplexType(s, ct.ComplexType)
		if base {
			sc.addBaseElement(ct.ComplexType.Name, typeName, ct.Doc)
		}
	}

	return typeName
}

// func getTypeName(in string) string {
// 	return makePublic(replaceReservedWords(stripns(in)))
// }

func (sc *schema) buildSimpleType(s *XSDSchema, t *XSDSimpleType) string {
	if t == nil {
		return ""
	}

	if t.Restriction == nil {
		log.Println("WARNING - simple type without a restriction - whats the point? " + t.Name)
	}

	typeName := t.Name
	_, ok := sc.types[typeName]
	if ok {
		log.Println("WARNING - fst type name conflict " + typeName)
		return ""
	}

	sst := SimpleType{
		Name: typeName,
		Type: t.Restriction.Base,
		// Restrictions []TRestriction // TODO optional
	}

	sc.types[typeName] = sst

	return typeName
}

func (sc *schema) buildComplexType(s *XSDSchema, t *XSDComplexType) string {
	if t == nil {
		return ""
	}

	if t.Abstract {
		log.Println("abstract types not supported - skipping")
		return ""
	}

	typeName := t.Name
	_, ok := sc.types[typeName]
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
		ft := sc.buildSimpleType(s, ct.SimpleType)
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

	// TODO - what are the differences
	sc.addElems(s, t.All, &sct)
	sc.addElems(s, t.Choice, &sct)
	sc.addElems(s, t.Sequence, &sct)
	sc.addElems(s, t.SequenceChoice, &sct)

	sc.types[typeName] = sct

	return typeName
}

func (sc *schema) addElems(s *XSDSchema, els []*XSDElement, sct *Struct) {
	for _, ct := range els {
		ft := sc.buildElement(s, ct, false)
		if ft != "" {
			sct.Fields = append(sct.Fields, sc.makeField(s, ct, ft))
		}
	}
}

func (sc *schema) makeField(s *XSDSchema, ct *XSDElement, fType string) *Field {
	oe := ",omitempty"
	if ct.MinOccurs != "0" {
		oe = ""
	}

	log.Println("making field: ", fType)
	name := ct.Name
	if name == "" {
		name = ct.Ref // use a ref if name not set
	}
	tag := fmt.Sprintf("`xml:\"%s%s\"`", name, oe)

	log.Println("making field 2: ", fType)
	return &Field{
		Comment:  ct.Doc,
		Name:     name,
		Type:     fType,
		XMLTag:   tag,
		Optional: false,
		RefName:  ct.Ref,
	}
}

func (sc *schema) toGoType(xsdType string) string {

	// Handles name space, ie. xsd:string, xs:string
	r := strings.Split(xsdType, ":")

	t := r[0]

	if len(r) == 2 {
		t = r[1]
	}

	lt := strings.ToLower(t)
	value := xsd2GoTypes[lt]

	if value != "" {
		return value
	}

	// now lookup if it is a complex type then only use a pointer
	var ok bool
	l, ok := sc.types[t]

	if ok && l.isComplex() {
		return "*" + l.TypeString()
	}

	return replaceReservedWords(makePublic(t))
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
	"datetime":      "string", // todo add some class that marshals as xs.DateTime etc needs
	"date":          "string", // ditto
	"time":          "string", // ditto
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
