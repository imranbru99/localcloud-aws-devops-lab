package protocols

import (
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/imrandev/localcloud/gateway/pkg/context"
)

// ParseQueryRequest parses urlencoded request from body or URL query
func ParseQueryRequest(r *http.Request, rawBody []byte, ctx *context.RequestContext) error {
	ctx.Protocol = "query"

	// Parse URL query
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			ctx.ParsedParams[k] = v[0]
		}
	}

	// Parse body if form-urlencoded
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

	// Action resolution
	if act := ctx.ParsedParams["Action"]; act != "" {
		ctx.Action = act
	}

	return nil
}

// WriteQueryResponse writes an XML response wrapped with Result and ResponseMetadata
func WriteQueryResponse(w http.ResponseWriter, ctx *context.RequestContext, resultXML []byte, xmlns string) {
	w.Header().Set("Content-Type", "text/xml; charset=UTF-8")
	w.Header().Set("x-amzn-RequestId", ctx.RequestID)
	w.WriteHeader(http.StatusOK)

	action := ctx.Action
	if action == "" {
		action = "Operation"
	}

	w.Write([]byte(xml.Header))
	w.Write([]byte("<" + action + "Response xmlns=\"" + xmlns + "\">\n"))
	w.Write([]byte("  <" + action + "Result>\n"))
	w.Write(resultXML)
	w.Write([]byte("\n  </" + action + "Result>\n"))
	w.Write([]byte("  <ResponseMetadata>\n"))
	w.Write([]byte("    <RequestId>" + ctx.RequestID + "</RequestId>\n"))
	w.Write([]byte("  </ResponseMetadata>\n"))
	w.Write([]byte("</" + action + "Response>\n"))
}

func ReadAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}
