// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gowsdl

var opsTmpl = `
{{range .}}
	{{$rawPortType := .Name}}
	{{$portType := .Name | makePublic}}
	type {{$portType}} struct {
		client *SOAPClient
	}

	func New{{$portType}}(url string, tls bool, auth *BasicAuth) *{{$portType}} {
		if url == "" {
			url = {{findServiceAddress .Name | printf "%q"}}
		}
		client := NewSOAPClient(url, tls, auth)

		return &{{$portType}}{
			client: client,
		}
	}

	{{range .Operations}}
		{{$faults := len .Faults}}
		{{$requestTypeName := findTypeFromMessage .Input.Message }}
		{{$soapAction := findSOAPAction .Name $rawPortType}}
		{{$responseTypeName := findTypeFromMessage .Output.Message }}

		{{if gt $faults 0}}
		// Error can be either of the following types:
		// {{range .Faults}}
		//   - {{.Name}} {{.Doc}}{{end}}{{end}}
		{{if ne .Doc ""}}/* {{.Doc}} */{{end}}
		func (service *{{$portType}}) {{makePublic .Name | replaceReservedWords}} ({{if ne $requestTypeName.Type ""}}request *{{$requestTypeName.Type}}{{end}}) (*{{$responseTypeName.Type}}, error) {
			response := new({{$responseTypeName.Type}})

			{{if ne $requestTypeName.Name ""}}
			// make sure we use any element referenced name
			request.XMLName = xml.Name{ Local: "{{$requestTypeName.Name}}" } 
			{{end}}

			err := service.client.Call("{{$soapAction}}", {{if ne $requestTypeName.Type ""}}request{{else}}nil{{end}}, response)
			if err != nil {
				return nil, err
			}

			return response, nil
		}

	{{end}}
{{end}}
`
