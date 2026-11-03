package eventbridge

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/imrandev/localcloud/gateway/pkg/context"
	"github.com/imrandev/localcloud/services"
)

type EventBridgeAdapter struct {
	mu    sync.RWMutex
	rules map[string]string
}

func init() {
	services.Register(&EventBridgeAdapter{
		rules: make(map[string]string),
	})
}

func (a *EventBridgeAdapter) Name() string {
	return "events"
}

func (a *EventBridgeAdapter) Protocol() services.Protocol {
	return services.ProtocolJSON11
}

func (a *EventBridgeAdapter) Operations() map[string]services.Handler {
	return map[string]services.Handler{
		"PutEvents": a.PutEvents,
		"PutRule":   a.PutRule,
		"ListRules": a.ListRules,
	}
}

func (a *EventBridgeAdapter) PutEvents(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	return map[string]interface{}{
		"FailedEntryCount": 0,
		"Entries": []map[string]string{
			{"EventId": uuid.New().String()},
		},
	}, nil
}

func (a *EventBridgeAdapter) PutRule(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var req struct {
		Name         string `json:"Name"`
		EventPattern string `json:"EventPattern"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	arn := fmt.Sprintf("arn:aws:events:%s:%s:rule/%s", ctx.Region, ctx.AccountID, req.Name)
	a.rules[req.Name] = arn
	return map[string]string{"RuleArn": arn}, nil
}

func (a *EventBridgeAdapter) ListRules(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var list []map[string]string
	for name, arn := range a.rules {
		list = append(list, map[string]string{"Name": name, "Arn": arn})
	}
	return map[string]interface{}{"Rules": list}, nil
}

func (a *EventBridgeAdapter) ResourceTypes() []services.ResourceType {
	return []services.ResourceType{
		{Name: "Rule", ARNFormat: "arn:aws:events:{region}:{account}:rule/{name}", Tags: true, Columns: []string{"Name", "Arn"}},
	}
}

func (a *EventBridgeAdapter) IAMActions() []services.ActionDef {
	return []services.ActionDef{
		{Name: "events:PutEvents", ResourceARNs: []string{"*"}},
	}
}

func (a *EventBridgeAdapter) HealthCheck() services.Health {
	return services.Health{Status: "healthy"}
}

func (a *EventBridgeAdapter) EmitEvents() []services.EventDef {
	return []services.EventDef{{Name: "PutEvents", Source: "events.amazonaws.com"}}
}

func (a *EventBridgeAdapter) Metrics() []services.MetricDef {
	return []services.MetricDef{}
}

func (a *EventBridgeAdapter) Quotas() []services.Quota {
	return []services.Quota{}
}

func (a *EventBridgeAdapter) Reset(account, region string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.rules = make(map[string]string)
	return nil
}

func (a *EventBridgeAdapter) Snapshot() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (a *EventBridgeAdapter) Restore(data map[string]interface{}) error {
	return nil
}
