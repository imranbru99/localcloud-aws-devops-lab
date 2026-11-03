package sqs

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/imrandev/localcloud/gateway/pkg/context"
	"github.com/imrandev/localcloud/services"
)

type SQSAdapter struct {
	mu     sync.RWMutex
	queues map[string]*Queue
}

type Message struct {
	MessageId     string `json:"MessageId"`
	ReceiptHandle string `json:"ReceiptHandle"`
	Body          string `json:"Body"`
	MD5OfBody     string `json:"MD5OfBody"`
	SentTime      time.Time `json:"-"`
}

type Queue struct {
	Name       string            `json:"name"`
	URL        string            `json:"url"`
	Arn        string            `json:"arn"`
	Messages   []*Message        `json:"messages"`
	Attributes map[string]string `json:"attributes"`
}

var DefaultSQS = &SQSAdapter{
	queues: make(map[string]*Queue),
}

func init() {
	services.Register(DefaultSQS)
}

func GetQueues() []*Queue {
	DefaultSQS.mu.RLock()
	defer DefaultSQS.mu.RUnlock()
	var list []*Queue
	for _, q := range DefaultSQS.queues {
		list = append(list, q)
	}
	return list
}

func (a *SQSAdapter) Name() string {
	return "sqs"
}

func (a *SQSAdapter) Protocol() services.Protocol {
	return services.ProtocolQuery
}

func (a *SQSAdapter) Operations() map[string]services.Handler {
	return map[string]services.Handler{
		"CreateQueue":        a.CreateQueue,
		"GetQueueUrl":        a.GetQueueUrl,
		"ListQueues":         a.ListQueues,
		"SendMessage":        a.SendMessage,
		"ReceiveMessage":     a.ReceiveMessage,
		"DeleteMessage":      a.DeleteMessage,
		"DeleteQueue":        a.DeleteQueue,
		"PurgeQueue":         a.PurgeQueue,
		"GetQueueAttributes": a.GetQueueAttributes,
	}
}

func (a *SQSAdapter) CreateQueue(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	queueName := ctx.ParsedParams["QueueName"]
	if queueName == "" && len(ctx.RawBody) > 0 {
		var req struct {
			QueueName string `json:"QueueName"`
		}
		_ = json.Unmarshal(ctx.RawBody, &req)
		queueName = req.QueueName
	}
	if queueName == "" {
		return nil, fmt.Errorf("QueueName parameter is required")
	}

	queueUrl := fmt.Sprintf("http://localhost:4566/%s/%s", ctx.AccountID, queueName)
	arn := fmt.Sprintf("arn:aws:sqs:%s:%s:%s", ctx.Region, ctx.AccountID, queueName)

	a.queues[queueName] = &Queue{
		Name:     queueName,
		URL:      queueUrl,
		Arn:      arn,
		Messages: make([]*Message, 0),
		Attributes: map[string]string{
			"QueueArn":                    arn,
			"ApproximateNumberOfMessages": "0",
			"VisibilityTimeout":           "30",
		},
	}

	if isJSONClient(r, ctx) {
		ctx.Protocol = "json10"
		return map[string]interface{}{"QueueUrl": queueUrl}, nil
	}

	xml := fmt.Sprintf(`    <QueueUrl>%s</QueueUrl>`, queueUrl)
	return []byte(xml), nil
}

func (a *SQSAdapter) GetQueueUrl(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	queueName := ctx.ParsedParams["QueueName"]
	if queueName == "" && len(ctx.RawBody) > 0 {
		var req struct {
			QueueName string `json:"QueueName"`
		}
		_ = json.Unmarshal(ctx.RawBody, &req)
		queueName = req.QueueName
	}

	q, ok := a.queues[queueName]
	if !ok {
		return nil, fmt.Errorf("AWS.SimpleQueueService.NonExistentQueue: The specified queue does not exist")
	}

	if isJSONClient(r, ctx) {
		ctx.Protocol = "json10"
		return map[string]interface{}{"QueueUrl": q.URL}, nil
	}

	xml := fmt.Sprintf(`    <QueueUrl>%s</QueueUrl>`, q.URL)
	return []byte(xml), nil
}

func (a *SQSAdapter) ListQueues(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	prefix := ctx.ParsedParams["QueueNamePrefix"]
	var urls []string
	var urlsXML string
	for name, q := range a.queues {
		if prefix == "" || strings.HasPrefix(name, prefix) {
			urls = append(urls, q.URL)
			urlsXML += fmt.Sprintf(`    <QueueUrl>%s</QueueUrl>
`, q.URL)
		}
	}

	if isJSONClient(r, ctx) {
		ctx.Protocol = "json10"
		return map[string]interface{}{"QueueUrls": urls}, nil
	}

	return []byte(urlsXML), nil
}

func (a *SQSAdapter) SendMessage(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	queueUrl := ctx.ParsedParams["QueueUrl"]
	msgBody := ctx.ParsedParams["MessageBody"]
	if msgBody == "" && len(ctx.RawBody) > 0 {
		var req struct {
			QueueUrl    string `json:"QueueUrl"`
			MessageBody string `json:"MessageBody"`
		}
		_ = json.Unmarshal(ctx.RawBody, &req)
		queueUrl = req.QueueUrl
		msgBody = req.MessageBody
	}

	queueName := extractQueueName(queueUrl, r.URL.Path)
	q, ok := a.queues[queueName]
	if !ok {
		q = &Queue{
			Name:     queueName,
			URL:      fmt.Sprintf("http://localhost:4566/%s/%s", ctx.AccountID, queueName),
			Arn:      fmt.Sprintf("arn:aws:sqs:%s:%s:%s", ctx.Region, ctx.AccountID, queueName),
			Messages: make([]*Message, 0),
		}
		a.queues[queueName] = q
	}

	hasher := md5.New()
	hasher.Write([]byte(msgBody))
	md5Str := hex.EncodeToString(hasher.Sum(nil))
	msgId := uuid.New().String()
	receipt := uuid.New().String()

	q.Messages = append(q.Messages, &Message{
		MessageId:     msgId,
		ReceiptHandle: receipt,
		Body:          msgBody,
		MD5OfBody:     md5Str,
		SentTime:      time.Now().UTC(),
	})

	if isJSONClient(r, ctx) {
		ctx.Protocol = "json10"
		return map[string]interface{}{
			"MD5OfMessageBody": md5Str,
			"MessageId":        msgId,
		}, nil
	}

	xml := fmt.Sprintf(`    <MD5OfMessageBody>%s</MD5OfMessageBody>
    <MessageId>%s</MessageId>`, md5Str, msgId)

	return []byte(xml), nil
}

func (a *SQSAdapter) ReceiveMessage(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	queueUrl := ctx.ParsedParams["QueueUrl"]
	if queueUrl == "" && len(ctx.RawBody) > 0 {
		var req struct {
			QueueUrl string `json:"QueueUrl"`
		}
		_ = json.Unmarshal(ctx.RawBody, &req)
		queueUrl = req.QueueUrl
	}

	queueName := extractQueueName(queueUrl, r.URL.Path)
	q, ok := a.queues[queueName]
	if !ok || len(q.Messages) == 0 {
		if isJSONClient(r, ctx) {
			ctx.Protocol = "json10"
			return map[string]interface{}{"Messages": []interface{}{}}, nil
		}
		return []byte(""), nil
	}

	msg := q.Messages[0]
	if isJSONClient(r, ctx) {
		ctx.Protocol = "json10"
		return map[string]interface{}{
			"Messages": []*Message{msg},
		}, nil
	}

	var msgsXML string
	msgsXML += fmt.Sprintf(`    <Message>
      <MessageId>%s</MessageId>
      <ReceiptHandle>%s</ReceiptHandle>
      <MD5OfBody>%s</MD5OfBody>
      <Body>%s</Body>
    </Message>`, msg.MessageId, msg.ReceiptHandle, msg.MD5OfBody, msg.Body)

	return []byte(msgsXML), nil
}

func (a *SQSAdapter) DeleteMessage(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	receipt := ctx.ParsedParams["ReceiptHandle"]
	queueUrl := ctx.ParsedParams["QueueUrl"]
	if receipt == "" && len(ctx.RawBody) > 0 {
		var req struct {
			ReceiptHandle string `json:"ReceiptHandle"`
			QueueUrl      string `json:"QueueUrl"`
		}
		_ = json.Unmarshal(ctx.RawBody, &req)
		receipt = req.ReceiptHandle
		queueUrl = req.QueueUrl
	}

	queueName := extractQueueName(queueUrl, r.URL.Path)
	if q, ok := a.queues[queueName]; ok {
		var filtered []*Message
		for _, m := range q.Messages {
			if m.ReceiptHandle != receipt {
				filtered = append(filtered, m)
			}
		}
		q.Messages = filtered
	}

	if isJSONClient(r, ctx) {
		ctx.Protocol = "json10"
		return map[string]interface{}{}, nil
	}
	return []byte(""), nil
}

func (a *SQSAdapter) DeleteQueue(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	queueUrl := ctx.ParsedParams["QueueUrl"]
	queueName := extractQueueName(queueUrl, r.URL.Path)
	delete(a.queues, queueName)

	if isJSONClient(r, ctx) {
		ctx.Protocol = "json10"
		return map[string]interface{}{}, nil
	}
	return []byte(""), nil
}

func (a *SQSAdapter) PurgeQueue(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	queueUrl := ctx.ParsedParams["QueueUrl"]
	queueName := extractQueueName(queueUrl, r.URL.Path)
	if q, ok := a.queues[queueName]; ok {
		q.Messages = make([]*Message, 0)
	}

	if isJSONClient(r, ctx) {
		ctx.Protocol = "json10"
		return map[string]interface{}{}, nil
	}
	return []byte(""), nil
}

func (a *SQSAdapter) GetQueueAttributes(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	queueUrl := ctx.ParsedParams["QueueUrl"]
	queueName := extractQueueName(queueUrl, r.URL.Path)
	q, ok := a.queues[queueName]
	if !ok {
		return []byte(""), nil
	}

	if isJSONClient(r, ctx) {
		ctx.Protocol = "json10"
		return map[string]interface{}{
			"Attributes": map[string]string{
				"QueueArn":                    q.Arn,
				"ApproximateNumberOfMessages": fmt.Sprintf("%d", len(q.Messages)),
			},
		}, nil
	}

	xml := fmt.Sprintf(`    <Attribute>
      <Name>QueueArn</Name>
      <Value>%s</Value>
    </Attribute>
    <Attribute>
      <Name>ApproximateNumberOfMessages</Name>
      <Value>%d</Value>
    </Attribute>`, q.Arn, len(q.Messages))

	return []byte(xml), nil
}

func isJSONClient(r *http.Request, ctx *context.RequestContext) bool {
	ct := r.Header.Get("Content-Type")
	target := r.Header.Get("X-Amz-Target")
	return strings.Contains(ct, "json") || strings.HasPrefix(target, "AmazonSQS") || ctx.Protocol == "json10" || ctx.Protocol == "json11"
}

func extractQueueName(queueUrl, path string) string {
	if queueUrl != "" {
		parts := strings.Split(queueUrl, "/")
		return parts[len(parts)-1]
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return parts[len(parts)-1]
}

func (a *SQSAdapter) ResourceTypes() []services.ResourceType {
	return []services.ResourceType{
		{Name: "Queue", ARNFormat: "arn:aws:sqs:{region}:{account}:{queuename}", Tags: true, Columns: []string{"Name", "URL", "ApproximateMessages"}},
	}
}

func (a *SQSAdapter) IAMActions() []services.ActionDef {
	return []services.ActionDef{
		{Name: "sqs:CreateQueue", ResourceARNs: []string{"arn:aws:sqs:*:*:*"}},
		{Name: "sqs:SendMessage", ResourceARNs: []string{"arn:aws:sqs:*:*:*"}},
		{Name: "sqs:ReceiveMessage", ResourceARNs: []string{"arn:aws:sqs:*:*:*"}},
		{Name: "sqs:DeleteMessage", ResourceARNs: []string{"arn:aws:sqs:*:*:*"}},
	}
}

func (a *SQSAdapter) HealthCheck() services.Health {
	return services.Health{Status: "healthy"}
}

func (a *SQSAdapter) EmitEvents() []services.EventDef {
	return []services.EventDef{
		{Name: "SendMessage", Source: "sqs.amazonaws.com"},
		{Name: "ReceiveMessage", Source: "sqs.amazonaws.com"},
	}
}

func (a *SQSAdapter) Metrics() []services.MetricDef {
	return []services.MetricDef{
		{Name: "NumberOfMessagesSent", Namespace: "AWS/SQS", Unit: "Count"},
		{Name: "ApproximateNumberOfMessagesVisible", Namespace: "AWS/SQS", Unit: "Count"},
	}
}

func (a *SQSAdapter) Quotas() []services.Quota {
	return []services.Quota{}
}

func (a *SQSAdapter) Reset(account, region string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.queues = make(map[string]*Queue)
	return nil
}

func (a *SQSAdapter) Snapshot() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (a *SQSAdapter) Restore(data map[string]interface{}) error {
	return nil
}
