package protocols

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/imrandev/localcloud/gateway/pkg/context"
)

func ParseJSONRequest(r *http.Request, rawBody []byte, ctx *context.RequestContext) error {
	ctx.Protocol = "json11"

	// Resolve from X-Amz-Target header: e.g. DynamoDB_20120810.CreateTable or TrentService.CreateKey
	target := r.Header.Get("X-Amz-Target")
	if target != "" {
		parts := strings.Split(target, ".")
		if len(parts) == 2 {
			ctx.Action = parts[1]
			svcParts := strings.Split(parts[0], "_")
			ctx.Service = strings.ToLower(svcParts[0])
		} else {
			ctx.Action = target
		}
	}

	if len(rawBody) > 0 {
		var bodyMap map[string]any
		if err := json.Unmarshal(rawBody, &bodyMap); err == nil {
			for k, v := range bodyMap {
				if str, ok := v.(string); ok {
					ctx.ParsedParams[k] = str
				}
			}
		}
	}

	return nil
}

func WriteJSONResponse(w http.ResponseWriter, ctx *context.RequestContext, data any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.Header().Set("x-amzn-RequestId", ctx.RequestID)
	w.WriteHeader(http.StatusOK)
	if data == nil {
		data = map[string]any{}
	}
	_ = json.NewEncoder(w).Encode(data)
}
