// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gowsdl

var soapTmpl = `
var timeout = time.Duration(30 * time.Second)

func dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

type SOAPEnvelope struct {
	XMLName xml.Name ` + "`" + `xml:"soapenv:Envelope"` + "`" + `

	SoapNs string ` + "`" + `xml:"xmlns:soapenv,attr"` + "`" + ` // simulate a prefixed namespace

	Body SOAPBody
}

type SOAPHeader struct {
	XMLName xml.Name ` + "`" + `xml:"soapenv:Header"` + "`" + `

	Header interface{}
}

type SOAPBody struct {
	XMLName xml.Name // set default namespace in the call

	Fault   *SOAPFault ` + "`" + `xml:",omitempty"` + "`" + `
	Content interface{} ` + "`" + `xml:",omitempty"` + "`" + `
}

type SOAPFault struct {
	XMLName xml.Name ` + "`" + `xml:"soapenv:Fault"` + "`" + `

	Code   string ` + "`" + `xml:"faultcode,omitempty"` + "`" + `
	String string ` + "`" + `xml:"faultstring,omitempty"` + "`" + `
	Actor  string ` + "`" + `xml:"faultactor,omitempty"` + "`" + `
	Detail string ` + "`" + `xml:"detail,omitempty"` + "`" + `
}

type BasicAuth struct {
	Login string
	Password string
}

type SOAPClient struct {
	url string
	tls bool
	auth *BasicAuth
}

func (b *SOAPBody) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if b.Content == nil {
		return xml.UnmarshalError("Content must be a pointer to a struct")
	}

	var (
		token    xml.Token
		err      error
		consumed bool
	)

Loop:
	for {
		if token, err = d.Token(); err != nil {
			return err
		}

		if token == nil {
			break
		}

		switch se := token.(type) {
		case xml.StartElement:
			if consumed {
				return xml.UnmarshalError("Found multiple elements inside SOAP body; not wrapped-document/literal WS-I compliant")
			} else if se.Name.Space == "http://schemas.xmlsoap.org/soap/envelope/" && se.Name.Local == "Fault" {
				b.Fault = &SOAPFault{}
				b.Content = nil

				err = d.DecodeElement(b.Fault, &se)
				if err != nil {
					return err
				}

				consumed = true
			} else {
				if err = d.DecodeElement(b.Content, &se); err != nil {
					return err
				}

				consumed = true
			}
		case xml.EndElement:
			break Loop
		}
	}

	return nil
}

func (f *SOAPFault) Error() string {
	return f.String
}

func NewSOAPClient(url string, tls bool, auth *BasicAuth) *SOAPClient {
	return &SOAPClient{
		url: url,
		tls: tls,
		auth: auth,
	}
}

// pass in the namespace as tns
func (s *SOAPClient) Call(soapAction string, request, response interface{}, tns string) error {
	envelope := SOAPEnvelope{
		SoapNs: "http://schemas.xmlsoap.org/soap/envelope/", // force our soapenv prefix namespace
	    // Header:        SoapHeader{},                      // TODO: face a real scenario that needs a header and implement it then 
	}

	// set the default namespace to that passed in
	envelope.Body.XMLName = xml.Name{Local: "soapenv:Body", Space: tns}
	envelope.Body.Content = request
	buffer := new(bytes.Buffer)

	encoder := xml.NewEncoder(buffer)
	encoder.Indent("", "    ")

	if err := encoder.Encode(envelope); err != nil {
		return err
	}

	if err := encoder.Flush(); err != nil {
		return err
	}

	log.Println(buffer.String())

	req, err := http.NewRequest("POST", s.url, buffer)
	if err != nil {
		return err
	}
	if s.auth != nil {
		req.SetBasicAuth(s.auth.Login, s.auth.Password)
	}

	req.Header.Add("Content-Type", "text/xml; charset=\"utf-8\"")
	if soapAction != "" {
		req.Header.Add("SOAPAction", soapAction)
	}

	req.Header.Set("User-Agent", "gowsdl/0.1")
	req.Close = true

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: s.tls,
		},
		Dial: dialTimeout,
	}

	client := &http.Client{Transport: tr}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	rawbody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if len(rawbody) == 0 {
		log.Println("empty response")
		return nil
	}

	log.Println(string(rawbody))
	respEnvelope := new(SOAPEnvelope)
	respEnvelope.Body = SOAPBody{Content: response}
	err = xml.Unmarshal(rawbody, respEnvelope)
	if err != nil {
		return err
	}

	fault := respEnvelope.Body.Fault
	if fault != nil {
		return fault
	}

	return nil
}
`
