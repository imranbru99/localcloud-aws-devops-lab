package sns

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/imrandev/localcloud/gateway/pkg/context"
	"github.com/imrandev/localcloud/services"
)

type SNSAdapter struct {
	mu            sync.RWMutex
	topics        map[string]*Topic
	subscriptions map[string]*Subscription
}

type Topic struct {
	Name string
	Arn  string
}

type Subscription struct {
	Arn      string
	TopicArn string
	Protocol string
	Endpoint string
}

func init() {
	services.Register(&SNSAdapter{
		topics:        make(map[string]*Topic),
		subscriptions: make(map[string]*Subscription),
	})
}

func (a *SNSAdapter) Name() string {
	return "sns"
}

func (a *SNSAdapter) Protocol() services.Protocol {
	return services.ProtocolQuery
}

func (a *SNSAdapter) Operations() map[string]services.Handler {
	return map[string]services.Handler{
		"CreateTopic":        a.CreateTopic,
		"ListTopics":         a.ListTopics,
		"DeleteTopic":        a.DeleteTopic,
		"Subscribe":          a.Subscribe,
		"Unsubscribe":        a.Unsubscribe,
		"ListSubscriptions":  a.ListSubscriptions,
		"Publish":            a.Publish,
	}
}

func (a *SNSAdapter) CreateTopic(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	topicName := ctx.ParsedParams["Name"]
	if topicName == "" {
		return nil, fmt.Errorf("Name parameter is required")
	}

	topicArn := fmt.Sprintf("arn:aws:sns:%s:%s:%s", ctx.Region, ctx.AccountID, topicName)
	a.topics[topicArn] = &Topic{
		Name: topicName,
		Arn:  topicArn,
	}

	xml := fmt.Sprintf(`    <TopicArn>%s</TopicArn>`, topicArn)
	return []byte(xml), nil
}

func (a *SNSAdapter) ListTopics(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var members string
	for _, t := range a.topics {
		members += fmt.Sprintf(`      <member>
        <TopicArn>%s</TopicArn>
      </member>
`, t.Arn)
	}

	xml := fmt.Sprintf(`    <Topics>
%s    </Topics>`, members)

	return []byte(xml), nil
}

func (a *SNSAdapter) DeleteTopic(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	topicArn := ctx.ParsedParams["TopicArn"]
	delete(a.topics, topicArn)
	return []byte(""), nil
}

func (a *SNSAdapter) Subscribe(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	topicArn := ctx.ParsedParams["TopicArn"]
	proto := ctx.ParsedParams["Protocol"]
	endpoint := ctx.ParsedParams["Endpoint"]

	subArn := fmt.Sprintf("%s:%s", topicArn, uuid.New().String())
	a.subscriptions[subArn] = &Subscription{
		Arn:      subArn,
		TopicArn: topicArn,
		Protocol: proto,
		Endpoint: endpoint,
	}

	xml := fmt.Sprintf(`    <SubscriptionArn>%s</SubscriptionArn>`, subArn)
	return []byte(xml), nil
}

func (a *SNSAdapter) Unsubscribe(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	subArn := ctx.ParsedParams["SubscriptionArn"]
	delete(a.subscriptions, subArn)
	return []byte(""), nil
}

func (a *SNSAdapter) ListSubscriptions(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var members string
	for _, sub := range a.subscriptions {
		members += fmt.Sprintf(`      <member>
        <Owner>%s</Owner>
        <Protocol>%s</Protocol>
        <Endpoint>%s</Endpoint>
        <SubscriptionArn>%s</SubscriptionArn>
        <TopicArn>%s</TopicArn>
      </member>
`, ctx.AccountID, sub.Protocol, sub.Endpoint, sub.Arn, sub.TopicArn)
	}

	xml := fmt.Sprintf(`    <Subscriptions>
%s    </Subscriptions>`, members)

	return []byte(xml), nil
}

func (a *SNSAdapter) Publish(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	msgId := uuid.New().String()
	xml := fmt.Sprintf(`    <MessageId>%s</MessageId>`, msgId)
	return []byte(xml), nil
}

func (a *SNSAdapter) ResourceTypes() []services.ResourceType {
	return []services.ResourceType{
		{Name: "Topic", ARNFormat: "arn:aws:sns:{region}:{account}:{topicname}", Tags: true, Columns: []string{"Name", "Arn"}},
	}
}

func (a *SNSAdapter) IAMActions() []services.ActionDef {
	return []services.ActionDef{
		{Name: "sns:CreateTopic", ResourceARNs: []string{"arn:aws:sns:*:*:*"}},
		{Name: "sns:Publish", ResourceARNs: []string{"arn:aws:sns:*:*:*"}},
	}
}

func (a *SNSAdapter) HealthCheck() services.Health {
	return services.Health{Status: "healthy"}
}

func (a *SNSAdapter) EmitEvents() []services.EventDef {
	return []services.EventDef{
		{Name: "Publish", Source: "sns.amazonaws.com"},
	}
}

func (a *SNSAdapter) Metrics() []services.MetricDef {
	return []services.MetricDef{}
}

func (a *SNSAdapter) Quotas() []services.Quota {
	return []services.Quota{}
}

func (a *SNSAdapter) Reset(account, region string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.topics = make(map[string]*Topic)
	a.subscriptions = make(map[string]*Subscription)
	return nil
}

func (a *SNSAdapter) Snapshot() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (a *SNSAdapter) Restore(data map[string]interface{}) error {
	return nil
}
