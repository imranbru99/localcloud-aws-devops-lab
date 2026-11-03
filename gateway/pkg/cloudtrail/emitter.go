package cloudtrail

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/imrandev/localcloud/gateway/pkg/context"
)

type Event struct {
	EventID            string            `json:"eventID"`
	EventTime          string            `json:"eventTime"`
	EventSource        string            `json:"eventSource"`
	EventName          string            `json:"eventName"`
	AwsRegion          string            `json:"awsRegion"`
	SourceIPAddress    string            `json:"sourceIPAddress"`
	UserAgent          string            `json:"userAgent"`
	RequestID          string            `json:"requestID"`
	UserIdentity       UserIdentity      `json:"userIdentity"`
	RequestParameters  map[string]any    `json:"requestParameters,omitempty"`
	ResponseElements   any               `json:"responseElements,omitempty"`
	ErrorCode          string            `json:"errorCode,omitempty"`
	ErrorMessage       string            `json:"errorMessage,omitempty"`
	LatencyMs          int64             `json:"latencyMs"`
	Explanation        string            `json:"explanation,omitempty"`
	EquivalentCLI      string            `json:"equivalentCLI,omitempty"`
	EquivalentTerraform string           `json:"equivalentTerraform,omitempty"`
}

type UserIdentity struct {
	Type        string `json:"type"`
	PrincipalID string `json:"principalId"`
	Arn         string `json:"arn"`
	AccountID   string `json:"accountId"`
	AccessKeyID string `json:"accessKeyId"`
}

type EventPipeline struct {
	mu         sync.RWMutex
	events     []*Event
	maxEvents  int
	logFile    *os.File
	listeners  map[chan *Event]bool
}

var Pipeline = NewPipeline(1000)

func NewPipeline(maxEvents int) *EventPipeline {
	dataDir := os.Getenv("LOCALCLOUD_DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	trailDir := filepath.Join(dataDir, "cloudtrail")
	_ = os.MkdirAll(trailDir, 0755)

	filePath := filepath.Join(trailDir, "events.jsonl")
	f, _ := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

	return &EventPipeline{
		events:    make([]*Event, 0, maxEvents),
		maxEvents: maxEvents,
		logFile:   f,
		listeners: make(map[chan *Event]bool),
	}
}

func (p *EventPipeline) Record(ctx *context.RequestContext, params map[string]any, resp any, errCode, errMsg string, latencyMs int64) *Event {
	p.mu.Lock()
	defer p.mu.Unlock()

	source := fmt.Sprintf("%s.amazonaws.com", ctx.Service)
	if ctx.Service == "" {
		source = "sts.amazonaws.com"
	}

	explanation := generateExplanation(ctx.Service, ctx.Action, params, errCode == "")
	cliCode := generateCLI(ctx.Service, ctx.Action, params)
	tfCode := generateTerraform(ctx.Service, ctx.Action, params)

	ev := &Event{
		EventID:         uuid.New().String(),
		EventTime:       time.Now().UTC().Format(time.RFC3339),
		EventSource:     source,
		EventName:       ctx.Action,
		AwsRegion:       ctx.Region,
		SourceIPAddress: ctx.ClientIP,
		UserAgent:       ctx.UserAgent,
		RequestID:       ctx.RequestID,
		UserIdentity: UserIdentity{
			Type:        "IAMUser",
			PrincipalID: ctx.AccessKeyID,
			Arn:         ctx.PrincipalARN,
			AccountID:   ctx.AccountID,
			AccessKeyID: ctx.AccessKeyID,
		},
		RequestParameters:   params,
		ResponseElements:    resp,
		ErrorCode:           errCode,
		ErrorMessage:        errMsg,
		LatencyMs:           latencyMs,
		Explanation:         explanation,
		EquivalentCLI:       cliCode,
		EquivalentTerraform: tfCode,
	}

	if len(p.events) >= p.maxEvents {
		p.events = p.events[1:]
	}
	p.events = append(p.events, ev)

	// Persist to jsonl
	if p.logFile != nil {
		if b, err := json.Marshal(ev); err == nil {
			_, _ = p.logFile.Write(append(b, '\n'))
		}
	}

	// Notify SSE listeners
	for ch := range p.listeners {
		select {
		case ch <- ev:
		default:
		}
	}

	return ev
}

func (p *EventPipeline) GetRecentEvents(limit int) []*Event {
	p.mu.RLock()
	defer p.mu.RUnlock()

	n := len(p.events)
	if limit > n || limit <= 0 {
		limit = n
	}
	res := make([]*Event, limit)
	copy(res, p.events[n-limit:])

	// Return newest first
	for i, j := 0, len(res)-1; i < j; i, j = i+1, j-1 {
		res[i], res[j] = res[j], res[i]
	}
	return res
}

func (p *EventPipeline) Subscribe() chan *Event {
	p.mu.Lock()
	defer p.mu.Unlock()

	ch := make(chan *Event, 100)
	p.listeners[ch] = true
	return ch
}

func (p *EventPipeline) Unsubscribe(ch chan *Event) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.listeners, ch)
	close(ch)
}

func generateExplanation(service, action string, params map[string]any, success bool) string {
	status := "successfully processed"
	if !success {
		status = "failed with an error"
	}
	switch service {
	case "sts":
		if action == "GetCallerIdentity" {
			return fmt.Sprintf("Identity verification request %s. Evaluated calling credentials and returned current AWS account ID and caller ARN.", status)
		}
	case "s3":
		if b, ok := params["Bucket"]; ok {
			return fmt.Sprintf("S3 %s on bucket '%v' %s.", action, b, status)
		}
	case "iam":
		return fmt.Sprintf("IAM security control request '%s' %s.", action, status)
	case "sqs":
		return fmt.Sprintf("Message queue operation '%s' %s.", action, status)
	}
	return fmt.Sprintf("AWS service %s executed action '%s' (%s).", service, action, status)
}

func generateCLI(service, action string, params map[string]any) string {
	switch service {
	case "sts":
		if action == "GetCallerIdentity" {
			return "aws --endpoint-url http://localhost:4566 sts get-caller-identity"
		}
	case "s3":
		if action == "CreateBucket" {
			b := params["Bucket"]
			return fmt.Sprintf("aws --endpoint-url http://localhost:4566 s3 mb s3://%v", b)
		}
		if action == "ListBuckets" {
			return "aws --endpoint-url http://localhost:4566 s3 ls"
		}
	case "sqs":
		if action == "CreateQueue" {
			q := params["QueueName"]
			return fmt.Sprintf("aws --endpoint-url http://localhost:4566 sqs create-queue --queue-name %v", q)
		}
		if action == "ListQueues" {
			return "aws --endpoint-url http://localhost:4566 sqs list-queues"
		}
	}
	return fmt.Sprintf("aws --endpoint-url http://localhost:4566 %s %s", service, action)
}

func generateTerraform(service, action string, params map[string]any) string {
	switch service {
	case "s3":
		if action == "CreateBucket" {
			b := params["Bucket"]
			return fmt.Sprintf("resource \"aws_s3_bucket\" \"example\" {\n  bucket = \"%v\"\n}", b)
		}
	case "sqs":
		if action == "CreateQueue" {
			q := params["QueueName"]
			return fmt.Sprintf("resource \"aws_sqs_queue\" \"example\" {\n  name = \"%v\"\n}", q)
		}
	}
	return "# Terraform resource block for " + service + " " + action
}
