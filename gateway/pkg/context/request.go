package context

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type RequestContext struct {
	AccountID       string            `json:"account_id"`
	Region          string            `json:"region"`
	Service         string            `json:"service"`
	Action          string            `json:"action"`
	PrincipalARN    string            `json:"principal_arn"`
	AccessKeyID     string            `json:"access_key_id"`
	RequestID       string            `json:"request_id"`
	Timestamp       time.Time         `json:"timestamp"`
	ClientIP        string            `json:"client_ip"`
	UserAgent       string            `json:"user_agent"`
	Protocol        string            `json:"protocol"`
	ParsedParams    map[string]string `json:"parsed_params"`
	RawBody         []byte            `json:"-"`
	IsConsoleClient bool              `json:"is_console_client"`
	AuthError       string            `json:"auth_error,omitempty"`
	SigV4Valid      bool              `json:"sigv4_valid"`
}

func NewRequestContext(r *http.Request) *RequestContext {
	reqID := r.Header.Get("X-Amzn-Requestid")
	if reqID == "" {
		reqID = r.Header.Get("X-Amz-Request-Id")
	}
	if reqID == "" {
		reqID = uuid.New().String()
	}

	clientIP := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		clientIP = strings.Split(fwd, ",")[0]
	}

	return &RequestContext{
		AccountID:    "000000000000",
		Region:       "us-east-1",
		RequestID:    reqID,
		Timestamp:    time.Now().UTC(),
		ClientIP:     clientIP,
		UserAgent:    r.UserAgent(),
		ParsedParams: make(map[string]string),
	}
}
