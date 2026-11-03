package protocols

import (
	"encoding/xml"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/imrandev/localcloud/gateway/pkg/context"
)

func ParseRestXMLRequest(r *http.Request, rawBody []byte, ctx *context.RequestContext) error {
	ctx.Protocol = "rest-xml"
	ctx.Service = "s3"

	host := r.Host
	path := strings.TrimPrefix(r.URL.Path, "/")
	pathSegments := strings.Split(path, "/")

	var bucket, key string

	if strings.Contains(host, ".localhost") || strings.Contains(host, ".127.0.0.1") {
		parts := strings.Split(host, ".")
		if len(parts) > 2 && parts[0] != "s3" && parts[0] != "localhost" {
			bucket = parts[0]
			key = path
		}
	}

	if bucket == "" && len(pathSegments) > 0 && pathSegments[0] != "" {
		bucket = pathSegments[0]
		if len(pathSegments) > 1 {
			key = strings.Join(pathSegments[1:], "/")
		}
	}

	ctx.ParsedParams["Bucket"] = bucket
	ctx.ParsedParams["Key"] = key

	q := r.URL.Query()
	for k, v := range q {
		if len(v) > 0 {
			ctx.ParsedParams[k] = v[0]
		}
	}

	switch r.Method {
	case http.MethodGet:
		if bucket == "" {
			ctx.Action = "ListBuckets"
		} else if key == "" {
			if _, ok := q["versioning"]; ok {
				ctx.Action = "GetBucketVersioning"
			} else if _, ok := q["policy"]; ok {
				ctx.Action = "GetBucketPolicy"
			} else if _, ok := q["cors"]; ok {
				ctx.Action = "GetBucketCors"
			} else {
				ctx.Action = "ListObjectsV2"
			}
		} else {
			if _, ok := q["tagging"]; ok {
				ctx.Action = "GetObjectTagging"
			} else {
				ctx.Action = "GetObject"
			}
		}
	case http.MethodPut:
		if key == "" {
			if _, ok := q["versioning"]; ok {
				ctx.Action = "PutBucketVersioning"
			} else if _, ok := q["policy"]; ok {
				ctx.Action = "PutBucketPolicy"
			} else if _, ok := q["cors"]; ok {
				ctx.Action = "PutBucketCors"
			} else {
				ctx.Action = "CreateBucket"
			}
		} else {
			if _, ok := q["tagging"]; ok {
				ctx.Action = "PutObjectTagging"
			} else {
				ctx.Action = "PutObject"
			}
		}
	case http.MethodHead:
		if key == "" {
			ctx.Action = "HeadBucket"
		} else {
			ctx.Action = "HeadObject"
		}
	case http.MethodDelete:
		if key == "" {
			if _, ok := q["policy"]; ok {
				ctx.Action = "DeleteBucketPolicy"
			} else {
				ctx.Action = "DeleteBucket"
			}
		} else {
			ctx.Action = "DeleteObject"
		}
	case http.MethodPost:
		if _, ok := q["delete"]; ok {
			ctx.Action = "DeleteObjects"
		}
	}

	return nil
}

func WriteRestXMLResponse(w http.ResponseWriter, ctx *context.RequestContext, data any) {
	w.Header().Set("x-amz-request-id", ctx.RequestID)
	w.Header().Set("x-amz-id-2", "localcloud-s3-host")

	if ctx.Action == "CreateBucket" {
		w.Header().Set("Location", "/"+ctx.ParsedParams["Bucket"])
		w.WriteHeader(http.StatusOK)
		return
	}
	if ctx.Action == "PutObject" {
		w.Header().Set("ETag", "\"d41d8cd98f00b204e9800998ecf8427e\"")
		w.WriteHeader(http.StatusOK)
		return
	}
	if ctx.Action == "HeadObject" {
		if size, ok := data.(int64); ok {
			w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("ETag", "\"d41d8cd98f00b204e9800998ecf8427e\"")
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(http.StatusOK)
		return
	}
	if ctx.Action == "HeadBucket" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if b, ok := data.([]byte); ok {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(len(b)))
		w.Header().Set("ETag", "\"d41d8cd98f00b204e9800998ecf8427e\"")
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(http.StatusOK)
		w.Write(b)
		return
	}
	if data == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(xml.Header))
	_ = xml.NewEncoder(w).Encode(data)
}
