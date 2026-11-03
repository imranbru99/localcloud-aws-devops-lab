package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/imrandev/localcloud/gateway/pkg/context"
)

type SigV4Parsed struct {
	Algorithm     string
	AccessKeyID   string
	Date          string
	Region        string
	Service       string
	RequestType   string
	SignedHeaders []string
	Signature     string
}

// CredentialsStore stores local access keys and secrets
type CredentialsStore struct {
	keys map[string]string // AccessKeyId -> SecretAccessKey
}

var DefaultCredentials = &CredentialsStore{
	keys: map[string]string{
		"test":                 "test",
		"localcloud":           "localcloud",
		"AKIAIOSFODNN7EXAMPLE": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"ROOT":                 "localcloud-root-secret",
	},
}

func (cs *CredentialsStore) GetSecret(accessKeyID string) (string, bool) {
	secret, exists := cs.keys[accessKeyID]
	if exists {
		return secret, true
	}
	// For local learning environments, allow arbitrary test keys with identical secret or standard dummy
	if strings.HasPrefix(accessKeyID, "AKIA") || strings.HasPrefix(accessKeyID, "ASIA") || accessKeyID != "" {
		return "test", true
	}
	return "", false
}

func (cs *CredentialsStore) RegisterKey(accessKeyID, secretKey string) {
	cs.keys[accessKeyID] = secretKey
}

func ParseSigV4(r *http.Request) (*SigV4Parsed, error) {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "AWS4-HMAC-SHA256 ") {
		return parseAuthHeader(authHeader)
	}

	// Check query string presigned
	q := r.URL.Query()
	if q.Get("X-Amz-Algorithm") == "AWS4-HMAC-SHA256" {
		cred := q.Get("X-Amz-Credential")
		parts := strings.Split(cred, "/")
		if len(parts) >= 5 {
			signedHeaders := strings.Split(q.Get("X-Amz-SignedHeaders"), ";")
			return &SigV4Parsed{
				Algorithm:     "AWS4-HMAC-SHA256",
				AccessKeyID:   parts[0],
				Date:          parts[1],
				Region:        parts[2],
				Service:       parts[3],
				RequestType:   parts[4],
				SignedHeaders: signedHeaders,
				Signature:     q.Get("X-Amz-Signature"),
			}, nil
		}
	}

	return nil, fmt.Errorf("no SigV4 authorization found")
}

func parseAuthHeader(header string) (*SigV4Parsed, error) {
	trimmed := strings.TrimPrefix(header, "AWS4-HMAC-SHA256 ")
	parts := strings.Split(trimmed, ",")
	res := &SigV4Parsed{Algorithm: "AWS4-HMAC-SHA256"}

	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) != 2 {
			continue
		}
		key, val := kv[0], kv[1]
		switch key {
		case "Credential":
			credParts := strings.Split(val, "/")
			if len(credParts) >= 5 {
				res.AccessKeyID = credParts[0]
				res.Date = credParts[1]
				res.Region = credParts[2]
				res.Service = credParts[3]
				res.RequestType = credParts[4]
			}
		case "SignedHeaders":
			res.SignedHeaders = strings.Split(val, ";")
		case "Signature":
			res.Signature = val
		}
	}

	if res.AccessKeyID == "" || res.Signature == "" {
		return nil, fmt.Errorf("malformed SigV4 authorization header")
	}

	return res, nil
}

func hmacSHA256(key []byte, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func VerifySigV4(r *http.Request, body []byte, cs *CredentialsStore) (*SigV4Parsed, bool, error) {
	parsed, err := ParseSigV4(r)
	if err != nil {
		// If no header, allow anonymous/dev mode for console internal calls
		return nil, false, err
	}

	secret, found := cs.GetSecret(parsed.AccessKeyID)
	if !found {
		return parsed, false, fmt.Errorf("invalid access key ID: %s", parsed.AccessKeyID)
	}

	// Build Canonical Request
	httpMethod := r.Method
	canonicalURI := r.URL.Path
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	// Canonical Query String
	canonicalQuery := buildCanonicalQuery(r.URL.Query())

	// Canonical Headers
	canonicalHeaders, signedHeadersStr := buildCanonicalHeaders(r, parsed.SignedHeaders)

	// Hashed Payload
	payloadHash := r.Header.Get("X-Amz-Content-Sha256")
	if payloadHash == "" {
		payloadHash = sha256Hex(body)
	}

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpMethod,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders,
		signedHeadersStr,
		payloadHash,
	)

	// Build StringToSign
	dateHeader := r.Header.Get("X-Amz-Date")
	if dateHeader == "" {
		dateHeader = r.Header.Get("Date")
	}
	if dateHeader == "" && r.URL.Query().Get("X-Amz-Date") != "" {
		dateHeader = r.URL.Query().Get("X-Amz-Date")
	}
	if dateHeader == "" {
		dateHeader = time.Now().UTC().Format("20060102T150405Z")
	}

	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", parsed.Date, parsed.Region, parsed.Service)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		dateHeader,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	)

	// Calculate Signing Key
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(parsed.Date))
	kRegion := hmacSHA256(kDate, []byte(parsed.Region))
	kService := hmacSHA256(kRegion, []byte(parsed.Service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))

	// Expected Signature
	expectedSignature := hex.EncodeToString(hmacSHA256(kSigning, []byte(stringToSign)))

	// Check if matches or if running in local education/permissive mode
	sigMatch := hmac.Equal([]byte(strings.ToLower(parsed.Signature)), []byte(strings.ToLower(expectedSignature)))
	return parsed, sigMatch, nil
}

func buildCanonicalQuery(values url.Values) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		if k == "X-Amz-Signature" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		escapedKey := url.QueryEscape(k)
		for _, v := range values[k] {
			escapedVal := url.QueryEscape(v)
			pairs = append(pairs, escapedKey+"="+escapedVal)
		}
	}
	return strings.Join(pairs, "&")
}

func buildCanonicalHeaders(r *http.Request, signedHeaders []string) (string, string) {
	headers := make(map[string]string)
	for k, vv := range r.Header {
		lowerKey := strings.ToLower(k)
		headers[lowerKey] = strings.TrimSpace(strings.Join(vv, ","))
	}
	// Ensure host is present
	if _, ok := headers["host"]; !ok && r.Host != "" {
		headers["host"] = r.Host
	}

	sort.Strings(signedHeaders)
	var canon strings.Builder
	for _, h := range signedHeaders {
		lh := strings.ToLower(h)
		canon.WriteString(lh)
		canon.WriteString(":")
		canon.WriteString(headers[lh])
		canon.WriteString("\n")
	}

	return canon.String(), strings.Join(signedHeaders, ";")
}

func AuthenticateRequest(r *http.Request, body []byte, ctx *context.RequestContext) {
	parsed, valid, err := VerifySigV4(r, body, DefaultCredentials)
	if err == nil && parsed != nil {
		ctx.AccessKeyID = parsed.AccessKeyID
		if parsed.Region != "" {
			ctx.Region = parsed.Region
		}
		if parsed.Service != "" {
			ctx.Service = parsed.Service
		}
		ctx.SigV4Valid = valid
		ctx.PrincipalARN = fmt.Sprintf("arn:aws:iam::%s:root", ctx.AccountID)
	} else {
		// Fallback for mock/test calls
		if r.Header.Get("X-Amz-Target") != "" {
			targetParts := strings.Split(r.Header.Get("X-Amz-Target"), ".")
			if len(targetParts) > 0 {
				ctx.Service = strings.ToLower(targetParts[0])
			}
		}
		ctx.AccessKeyID = "AKIAIOSFODNN7EXAMPLE"
		ctx.PrincipalARN = fmt.Sprintf("arn:aws:iam::%s:root", ctx.AccountID)
		ctx.SigV4Valid = true
	}
}
