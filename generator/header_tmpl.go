package generator

var headerTmpl = `
package {{.}}
//Generated by https://github.com/c4milo/gowsdl
//Do not modify
//Copyright (c) 2014, Camilo Aguilar. All rights reserved.
import (
	"encoding/xml"
	"time"
	"net"
	gowsdl "github.com/c4milo/gowsdl/generator"
	"net/http"
	"io/ioutil"
	"log"
	"bytes"
	"crypto/tls"
	{{/*range .Imports*/}}
		{{/*.*/}}
	{{/*end*/}}
)
`
