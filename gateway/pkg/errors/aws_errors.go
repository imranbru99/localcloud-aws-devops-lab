package errors

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/imrandev/localcloud/gateway/pkg/context"
)

type AWSError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Type       string `json:"type,omitempty"`
}

func (e *AWSError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewAWSError(status int, code, message string) *AWSError {
	errType := "Sender"
	if status >= 500 {
		errType = "Receiver"
	}
	return &AWSError{
		StatusCode: status,
		Code:       code,
		Message:    message,
		Type:       errType,
	}
}

// Common errors
var (
	ErrAccessDenied = &AWSError{
		StatusCode: http.StatusForbidden,
		Code:       "AccessDenied",
		Message:    "Access denied",
		Type:       "Sender",
	}
	ErrInvalidParameter = &AWSError{
		StatusCode: http.StatusBadRequest,
		Code:       "InvalidParameterValue",
		Message:    "An invalid or out-of-range value was supplied for the input parameter.",
		Type:       "Sender",
	}
	ErrMissingParameter = &AWSError{
		StatusCode: http.StatusBadRequest,
		Code:       "MissingParameter",
		Message:    "The request must contain the specified parameter.",
		Type:       "Sender",
	}
	ErrResourceNotFound = &AWSError{
		StatusCode: http.StatusNotFound,
		Code:       "ResourceNotFoundException",
		Message:    "The specified resource does not exist.",
		Type:       "Sender",
	}
	ErrNoSuchBucket = &AWSError{
		StatusCode: http.StatusNotFound,
		Code:       "NoSuchBucket",
		Message:    "The specified bucket does not exist",
		Type:       "Sender",
	}
	ErrNoSuchKey = &AWSError{
		StatusCode: http.StatusNotFound,
		Code:       "NoSuchKey",
		Message:    "The specified key does not exist.",
		Type:       "Sender",
	}
	ErrInternalFailure = &AWSError{
		StatusCode: http.StatusInternalServerError,
		Code:       "InternalFailure",
		Message:    "The request processing has failed due to an unknown error, exception or failure.",
		Type:       "Receiver",
	}
)

func WriteErrorResponse(w http.ResponseWriter, ctx *context.RequestContext, err *AWSError) {
	if err == nil {
		err = ErrInternalFailure
	}
	w.Header().Set("x-amzn-RequestId", ctx.RequestID)
	w.Header().Set("x-amz-request-id", ctx.RequestID)

	status := err.StatusCode
	if status == 0 {
		status = http.StatusBadRequest
	}

	switch ctx.Protocol {
	case "json10", "json11", "rest-json":
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.WriteHeader(status)
		resp := map[string]interface{}{
			"__type":  err.Code,
			"message": err.Message,
			"Message": err.Message,
		}
		_ = json.NewEncoder(w).Encode(resp)

	case "rest-xml":
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(status)
		type S3Error struct {
			XMLName   xml.Name `xml:"Error"`
			Code      string   `xml:"Code"`
			Message   string   `xml:"Message"`
			RequestId string   `xml:"RequestId"`
			HostId    string   `xml:"HostId"`
		}
		resp := S3Error{
			Code:      err.Code,
			Message:   err.Message,
			RequestId: ctx.RequestID,
			HostId:    "localcloud",
		}
		w.Write([]byte(xml.Header))
		_ = xml.NewEncoder(w).Encode(resp)

	default: // query, ec2
		w.Header().Set("Content-Type", "text/xml")
		w.WriteHeader(status)
		type QueryErrorDetail struct {
			Type    string `xml:"Type"`
			Code    string `xml:"Code"`
			Message string `xml:"Message"`
		}
		type QueryErrorResponse struct {
			XMLName   xml.Name         `xml:"ErrorResponse"`
			Xmlns     string           `xml:"xmlns,attr"`
			Error     QueryErrorDetail `xml:"Error"`
			RequestId string           `xml:"RequestId"`
		}
		xmlns := "https://sts.amazonaws.com/doc/2011-06-15/"
		if ctx.Service == "iam" {
			xmlns = "https://iam.amazonaws.com/doc/2010-05-08/"
		} else if ctx.Service == "sqs" {
			xmlns = "http://queue.amazonaws.com/doc/2012-11-05/"
		} else if ctx.Service == "sns" {
			xmlns = "http://sns.amazonaws.com/doc/2010-03-31/"
		}
		resp := QueryErrorResponse{
			Xmlns: xmlns,
			Error: QueryErrorDetail{
				Type:    err.Type,
				Code:    err.Code,
				Message: err.Message,
			},
			RequestId: ctx.RequestID,
		}
		w.Write([]byte(xml.Header))
		_ = xml.NewEncoder(w).Encode(resp)
	}
}
