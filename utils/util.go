package utils

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/ghettovoice/gosip/sip"
	"golang.org/x/net/html/charset"
)

func GetBranchID(msg sip.Message) sip.MaybeString {
	if viaHop, ok := msg.ViaHop(); ok {
		if branch, ok := viaHop.Params.Get("branch"); ok {
			return branch
		}
	}

	return nil
}

func GetIP(addr string) string {
	if strings.Contains(addr, ":") {
		return strings.Split(addr, ":")[0]
	}
	return ""
}

func GetPort(addr string) string {
	if strings.Contains(addr, ":") {
		return strings.Split(addr, ":")[1]
	}
	return ""
}

func BuildContactHeader(name string, from, to sip.Message, expires *sip.Expires) {
	name = strings.ToLower(name)
	for _, h := range from.GetHeaders(name) {
		AddParamsToContact(h.(*sip.ContactHeader), expires)
		to.AppendHeader(h.Clone())
	}
}

func BuildRequest(
	method sip.RequestMethod,
	from *sip.Address,
	to *sip.Address,
	contact *sip.Address,
	recipient sip.SipUri,
	routes []sip.Uri,
	callID *sip.CallID,
	contentType *sip.ContentType) (sip.Request, error) {

	builder := sip.NewRequestBuilder()

	builder.SetMethod(method)
	builder.SetFrom(from)
	builder.SetTo(to)
	if contact != nil {
		builder.SetContact(contact)
	}
	builder.SetRecipient(recipient.Clone())

	if len(routes) > 0 {
		builder.SetRoutes(routes)
	}

	if callID != nil {
		builder.SetCallID(callID)
	}

	if contentType != nil {
		builder.SetContentType(contentType)
	}

	req, err := builder.Build()
	if err != nil {
		return nil, err
	}

	return req, nil
}

func AddParamsToContact(contact *sip.ContactHeader, expires *sip.Expires) {
	if urn, ok := contact.Params.Get("+sip.instance"); ok {
		contact.Params.Add("+sip.instance", sip.String{Str: fmt.Sprintf(`"%s"`, urn)})
	}
	if expires != nil {
		contact.Params.Add("expires", sip.String{Str: fmt.Sprintf("%d", int(*expires))})
	}
}

// Render params to a string.
// Note that this does not escape special characters, this should already have been done before calling this method.
func SipParamsToString(params sip.Params, sep uint8) string {
	if params == nil {
		return ""
	}

	var buffer bytes.Buffer
	first := true

	for _, key := range params.Keys() {
		val, ok := params.Get(key)
		if !ok {
			continue
		}

		if !first {
			buffer.WriteString(fmt.Sprintf("%c", sep))
		}
		first = false

		buffer.WriteString(key)

		if val, ok := val.(sip.String); ok {
			buffer.WriteString(fmt.Sprintf("=%s", val.String()))
		}
	}

	return buffer.String()
}

// XMLDecode XMLDecode
func XMLDecode(data []byte, v interface{}) error {
	decoder := xml.NewDecoder(bytes.NewReader([]byte(data)))
	decoder.CharsetReader = charset.NewReaderLabel
	return decoder.Decode(v)
}
