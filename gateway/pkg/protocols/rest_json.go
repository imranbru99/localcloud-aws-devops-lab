package protocols

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/imrandev/localcloud/gateway/pkg/context"
)

func ParseRestJSONRequest(r *http.Request, rawBody []byte, ctx *context.RequestContext) error {
	ctx.Protocol = "rest-json"
	path := r.URL.Path

	// Lambda path patterns: e.g. /2015-03-31/functions/{FunctionName}/invocations
	if strings.Contains(path, "/functions/") {
		ctx.Service = "lambda"
		parts := strings.Split(path, "/")
		for i, p := range parts {
			if p == "functions" && i+1 < len(parts) {
				ctx.ParsedParams["FunctionName"] = parts[i+1]
				if i+2 < len(parts) && parts[i+2] == "invocations" {
					ctx.Action = "Invoke"
				} else {
					switch r.Method {
					case http.MethodGet:
						ctx.Action = "GetFunction"
					case http.MethodDelete:
						ctx.Action = "DeleteFunction"
					}
				}
				break
			}
		}
		if ctx.Action == "" {
			if r.Method == http.MethodPost {
				ctx.Action = "CreateFunction"
			} else if r.Method == http.MethodGet {
				ctx.Action = "ListFunctions"
			}
		}
	}

	return nil
}

func WriteRestJSONResponse(w http.ResponseWriter, ctx *context.RequestContext, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("x-amzn-RequestId", ctx.RequestID)
	w.WriteHeader(statusCode)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}
