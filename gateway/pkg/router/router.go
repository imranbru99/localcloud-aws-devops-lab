package router

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/imrandev/localcloud/gateway/pkg/auth"
	"github.com/imrandev/localcloud/gateway/pkg/cloudtrail"
	"github.com/imrandev/localcloud/gateway/pkg/context"
	"github.com/imrandev/localcloud/gateway/pkg/errors"
	"github.com/imrandev/localcloud/gateway/pkg/iam"
	"github.com/imrandev/localcloud/gateway/pkg/protocols"
	"github.com/imrandev/localcloud/services"
)

type Router struct {
	consoleHandler http.Handler
}

func NewRouter(consoleHandler http.Handler) *Router {
	return &Router{
		consoleHandler: consoleHandler,
	}
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Check if request is for Console UI or Internal REST API
	if strings.HasPrefix(r.URL.Path, "/console") ||
		strings.HasPrefix(r.URL.Path, "/api/") ||
		strings.HasPrefix(r.URL.Path, "/assets/") ||
		strings.HasPrefix(r.URL.Path, "/static/") ||
		r.URL.Path == "/favicon.ico" ||
		(r.URL.Path == "/" && r.Method == http.MethodGet && r.Header.Get("Authorization") == "" && r.Header.Get("X-Amz-Target") == "" && r.URL.Query().Get("Action") == "") {
		if r.URL.Path == "/" && strings.Contains(r.Header.Get("Accept"), "text/html") {
			http.Redirect(w, r, "/console#/dashboard", http.StatusFound)
			return
		}
		if rt.consoleHandler != nil {
			rt.consoleHandler.ServeHTTP(w, r)
			return
		}
	}

	start := time.Now()
	ctx := context.NewRequestContext(r)

	var rawBody []byte
	if r.Body != nil {
		rawBody, _ = io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewBuffer(rawBody))
		ctx.RawBody = rawBody
	}

	// 2. Identify service & protocol
	service, protocol := identifyService(r, rawBody)
	ctx.Service = service
	ctx.Protocol = string(protocol)

	// 3. Protocol Parser
	switch protocol {
	case services.ProtocolQuery:
		_ = protocols.ParseQueryRequest(r, rawBody, ctx)
	case services.ProtocolJSON10, services.ProtocolJSON11:
		_ = protocols.ParseJSONRequest(r, rawBody, ctx)
	case services.ProtocolRestXML:
		_ = protocols.ParseRestXMLRequest(r, rawBody, ctx)
	case services.ProtocolRestJSON:
		_ = protocols.ParseRestJSONRequest(r, rawBody, ctx)
	case services.ProtocolEC2:
		_ = protocols.ParseEC2QueryRequest(r, rawBody, ctx)
	}

	// 4. Authenticate caller (SigV4)
	auth.AuthenticateRequest(r, rawBody, ctx)

	// 5. Lookup Service Adapter
	adapter, exists := services.Get(ctx.Service)
	if !exists {
		if ctx.Service == "" || ctx.Service == "s3" {
			adapter, exists = services.Get("s3")
			ctx.Service = "s3"
			ctx.Protocol = "rest-xml"
		}
	}

	if !exists {
		errResp := errors.NewAWSError(http.StatusBadRequest, "InvalidAction", "Could not route request to AWS service: "+ctx.Service)
		errors.WriteErrorResponse(w, ctx, errResp)
		cloudtrail.Pipeline.Record(ctx, toAnyMap(ctx.ParsedParams), nil, errResp.Code, errResp.Message, time.Since(start).Milliseconds())
		return
	}

	// 6. Resolve operation handler
	handler, opExists := adapter.Operations()[ctx.Action]
	if !opExists {
		errResp := errors.NewAWSError(http.StatusBadRequest, "InvalidAction", "The action "+ctx.Action+" is not valid for service "+ctx.Service)
		errors.WriteErrorResponse(w, ctx, errResp)
		cloudtrail.Pipeline.Record(ctx, toAnyMap(ctx.ParsedParams), nil, errResp.Code, errResp.Message, time.Since(start).Milliseconds())
		return
	}

	// 7. IAM Policy Evaluation
	actionName := ctx.Service + ":" + ctx.Action
	resourceArn := "*"
	if b, ok := ctx.ParsedParams["Bucket"]; ok && b != "" {
		resourceArn = "arn:aws:s3:::" + b
	} else if q, ok := ctx.ParsedParams["QueueName"]; ok && q != "" {
		resourceArn = "arn:aws:sqs:" + ctx.Region + ":" + ctx.AccountID + ":" + q
	}

	eval := iam.Evaluate(ctx, actionName, resourceArn)
	if !eval.Allowed {
		errResp := errors.NewAWSError(http.StatusForbidden, "AccessDenied",
			"User: "+ctx.PrincipalARN+" is not authorized to perform: "+actionName+" on resource: "+resourceArn+" because "+eval.Reason)
		errors.WriteErrorResponse(w, ctx, errResp)
		cloudtrail.Pipeline.Record(ctx, toAnyMap(ctx.ParsedParams), nil, errResp.Code, errResp.Message, time.Since(start).Milliseconds())
		return
	}

	// 8. Execute Operation
	res, err := handler(ctx, r)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		awsErr, ok := err.(*errors.AWSError)
		if !ok {
			awsErr = errors.NewAWSError(http.StatusInternalServerError, "InternalFailure", err.Error())
		}
		errors.WriteErrorResponse(w, ctx, awsErr)
		cloudtrail.Pipeline.Record(ctx, toAnyMap(ctx.ParsedParams), nil, awsErr.Code, awsErr.Message, latency)
		return
	}

	// 9. Format response according to protocol
	writeResponse(w, ctx, services.Protocol(ctx.Protocol), res)
	cloudtrail.Pipeline.Record(ctx, toAnyMap(ctx.ParsedParams), res, "", "", latency)
}

func identifyService(r *http.Request, rawBody []byte) (string, services.Protocol) {
	// 1. X-Amz-Target header
	target := r.Header.Get("X-Amz-Target")
	if target != "" {
		parts := strings.Split(target, ".")
		svc := strings.ToLower(parts[0])
		if strings.HasPrefix(svc, "amazonsqs") || strings.HasPrefix(svc, "sqs") {
			return "sqs", services.ProtocolJSON10
		}
		if strings.HasPrefix(svc, "dynamodb") {
			return "dynamodb", services.ProtocolJSON10
		}
		if strings.HasPrefix(svc, "trentservice") || strings.Contains(svc, "kms") {
			return "kms", services.ProtocolJSON11
		}
		if strings.HasPrefix(svc, "amazonsm") || strings.Contains(svc, "secretsmanager") {
			return "secretsmanager", services.ProtocolJSON11
		}
		if strings.HasPrefix(svc, "amazonssm") || strings.Contains(svc, "ssm") {
			return "ssm", services.ProtocolJSON11
		}
		if strings.HasPrefix(svc, "logs") {
			return "logs", services.ProtocolJSON11
		}
		if strings.HasPrefix(svc, "awsevents") || strings.Contains(svc, "events") {
			return "events", services.ProtocolJSON11
		}
		return svc, services.ProtocolJSON11
	}

	// 2. Authorization header
	if parsed, err := auth.ParseSigV4(r); err == nil && parsed.Service != "" {
		switch parsed.Service {
		case "sts":
			return "sts", services.ProtocolQuery
		case "iam":
			return "iam", services.ProtocolQuery
		case "s3":
			return "s3", services.ProtocolRestXML
		case "sqs":
			if strings.Contains(r.Header.Get("Content-Type"), "json") {
				return "sqs", services.ProtocolJSON10
			}
			return "sqs", services.ProtocolQuery
		case "sns":
			return "sns", services.ProtocolQuery
		case "ses", "email":
			return "ses", services.ProtocolQuery
		case "cloudformation":
			return "cloudformation", services.ProtocolQuery
		case "rds":
			return "rds", services.ProtocolQuery
		case "route53":
			return "route53", services.ProtocolRestXML
		case "events":
			return "events", services.ProtocolJSON11
		case "ec2":
			return "ec2", services.ProtocolEC2
		case "monitoring":
			return "cloudwatch", services.ProtocolQuery
		case "lambda":
			return "lambda", services.ProtocolRestJSON
		}
	}

	// 3. Inspect Path and Query parameters
	path := r.URL.Path
	if strings.Contains(path, "/functions") {
		return "lambda", services.ProtocolRestJSON
	}
	if strings.Contains(path, "/hostedzone") {
		return "route53", services.ProtocolRestXML
	}

	queryAction := r.URL.Query().Get("Action")
	bodyStr := string(rawBody)

	if queryAction != "" || strings.Contains(bodyStr, "Action=") {
		action := queryAction
		if action == "" {
			for _, part := range strings.Split(bodyStr, "&") {
				if strings.HasPrefix(part, "Action=") {
					action = strings.TrimPrefix(part, "Action=")
					break
				}
			}
		}

		if strings.Contains(action, "Email") || strings.Contains(action, "Identity") {
			return "ses", services.ProtocolQuery
		}
		if strings.Contains(action, "Stack") {
			return "cloudformation", services.ProtocolQuery
		}
		if strings.Contains(action, "DBInstance") {
			return "rds", services.ProtocolQuery
		}
		if strings.HasPrefix(action, "DescribeInstances") || strings.HasPrefix(action, "RunInstances") ||
			strings.HasPrefix(action, "TerminateInstances") || strings.HasPrefix(action, "CreateSecurityGroup") ||
			strings.HasPrefix(action, "DescribeVpcs") {
			return "ec2", services.ProtocolEC2
		}
		if strings.Contains(action, "CallerIdentity") || strings.Contains(action, "AssumeRole") || strings.Contains(action, "SessionToken") {
			return "sts", services.ProtocolQuery
		}
		if strings.Contains(action, "User") || strings.Contains(action, "Role") || strings.Contains(action, "Policy") || strings.Contains(action, "AccessKey") {
			return "iam", services.ProtocolQuery
		}
		if strings.Contains(action, "Queue") || strings.Contains(action, "Message") {
			return "sqs", services.ProtocolQuery
		}
		if strings.Contains(action, "Topic") || strings.Contains(action, "Subscription") || strings.Contains(action, "Publish") {
			return "sns", services.ProtocolQuery
		}
		if strings.Contains(action, "Metric") || strings.Contains(action, "Alarm") {
			return "cloudwatch", services.ProtocolQuery
		}
	}

	if r.Header.Get("X-Amz-Content-Sha256") != "" || r.Method == http.MethodPut || r.Method == http.MethodDelete || r.Method == http.MethodHead {
		return "s3", services.ProtocolRestXML
	}

	return "s3", services.ProtocolRestXML
}

func writeResponse(w http.ResponseWriter, ctx *context.RequestContext, proto services.Protocol, res any) {
	switch proto {
	case services.ProtocolQuery:
		if xmlBytes, ok := res.([]byte); ok {
			xmlns := "https://sts.amazonaws.com/doc/2011-06-15/"
			switch ctx.Service {
			case "iam":
				xmlns = "https://iam.amazonaws.com/doc/2010-05-08/"
			case "sqs":
				xmlns = "http://queue.amazonaws.com/doc/2012-11-05/"
			case "sns":
				xmlns = "http://sns.amazonaws.com/doc/2010-03-31/"
			case "ses":
				xmlns = "http://ses.amazonaws.com/doc/2010-12-01/"
			case "cloudformation":
				xmlns = "http://cloudformation.amazonaws.com/doc/2010-05-15/"
			case "rds":
				xmlns = "http://rds.amazonaws.com/doc/2014-10-31/"
			case "cloudwatch":
				xmlns = "http://monitoring.amazonaws.com/doc/2010-08-01/"
			}
			protocols.WriteQueryResponse(w, ctx, xmlBytes, xmlns)
		}
	case services.ProtocolJSON10, services.ProtocolJSON11:
		protocols.WriteJSONResponse(w, ctx, res)
	case services.ProtocolRestXML:
		protocols.WriteRestXMLResponse(w, ctx, res)
	case services.ProtocolRestJSON:
		protocols.WriteRestJSONResponse(w, ctx, http.StatusOK, res)
	case services.ProtocolEC2:
		if xmlBytes, ok := res.([]byte); ok {
			protocols.WriteEC2Response(w, ctx, xmlBytes)
		}
	}
}

func toAnyMap(m map[string]string) map[string]any {
	out := make(map[string]any)
	for k, v := range m {
		out[k] = v
	}
	return out
}
