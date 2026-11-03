package protocols

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"

	"github.com/imrandev/localcloud/gateway/pkg/context"
)

func ParseEC2QueryRequest(r *http.Request, rawBody []byte, ctx *context.RequestContext) error {
	ctx.Protocol = "ec2"
	ctx.Service = "ec2"

	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			ctx.ParsedParams[k] = v[0]
		}
	}

	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "application/x-www-form-urlencoded") && len(rawBody) > 0 {
		vals, err := url.ParseQuery(string(rawBody))
		if err == nil {
			for k, v := range vals {
				if len(v) > 0 {
					ctx.ParsedParams[k] = v[0]
				}
			}
		}
	}

	if act := ctx.ParsedParams["Action"]; act != "" {
		ctx.Action = act
	}

	return nil
}

func WriteEC2Response(w http.ResponseWriter, ctx *context.RequestContext, resultXML []byte) {
	w.Header().Set("Content-Type", "text/xml; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	action := ctx.Action
	if action == "" {
		action = "Operation"
	}

	w.Write([]byte(xml.Header))
	w.Write([]byte("<" + action + "Response xmlns=\"http://ec2.amazonaws.com/doc/2016-11-15/\">\n"))
	w.Write([]byte("  <requestId>" + ctx.RequestID + "</requestId>\n"))
	w.Write(resultXML)
	w.Write([]byte("\n</" + action + "Response>\n"))
}
