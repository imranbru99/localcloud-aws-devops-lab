package ssm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/imrandev/localcloud/gateway/pkg/context"
	"github.com/imrandev/localcloud/services"
)

type SSMAdapter struct {
	mu     sync.RWMutex
	params map[string]*ParameterMeta
}

type ParameterMeta struct {
	Name             string  `json:"Name"`
	Type             string  `json:"Type"`
	Value            string  `json:"Value"`
	Version          int64   `json:"Version"`
	LastModifiedDate float64 `json:"LastModifiedDate"`
	ARN              string  `json:"ARN"`
}

func init() {
	services.Register(&SSMAdapter{
		params: make(map[string]*ParameterMeta),
	})
}

func (a *SSMAdapter) Name() string {
	return "ssm"
}

func (a *SSMAdapter) Protocol() services.Protocol {
	return services.ProtocolJSON11
}

func (a *SSMAdapter) Operations() map[string]services.Handler {
	return map[string]services.Handler{
		"PutParameter":        a.PutParameter,
		"GetParameter":        a.GetParameter,
		"GetParameters":       a.GetParameters,
		"GetParametersByPath": a.GetParametersByPath,
		"DeleteParameter":     a.DeleteParameter,
	}
}

func (a *SSMAdapter) PutParameter(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var req struct {
		Name      string `json:"Name"`
		Type      string `json:"Type"`
		Value     string `json:"Value"`
		Overwrite bool   `json:"Overwrite"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	if req.Name == "" {
		return nil, fmt.Errorf("Name is required")
	}
	if req.Type == "" {
		req.Type = "String"
	}

	p, exists := a.params[req.Name]
	var ver int64 = 1
	if exists {
		ver = p.Version + 1
	}

	arn := fmt.Sprintf("arn:aws:ssm:%s:%s:parameter%s", ctx.Region, ctx.AccountID, req.Name)
	newP := &ParameterMeta{
		Name:             req.Name,
		Type:             req.Type,
		Value:            req.Value,
		Version:          ver,
		LastModifiedDate: float64(time.Now().Unix()),
		ARN:              arn,
	}

	a.params[req.Name] = newP
	return map[string]interface{}{"Version": ver, "Tier": "Standard"}, nil
}

func (a *SSMAdapter) GetParameter(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var req struct {
		Name           string `json:"Name"`
		WithDecryption bool   `json:"WithDecryption"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	p, ok := a.params[req.Name]
	if !ok {
		return nil, fmt.Errorf("ParameterNotFound: %s", req.Name)
	}

	return map[string]interface{}{"Parameter": p}, nil
}

func (a *SSMAdapter) GetParameters(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var req struct {
		Names []string `json:"Names"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	var found []*ParameterMeta
	var invalid []string
	for _, name := range req.Names {
		if p, ok := a.params[name]; ok {
			found = append(found, p)
		} else {
			invalid = append(invalid, name)
		}
	}

	return map[string]interface{}{
		"Parameters":        found,
		"InvalidParameters": invalid,
	}, nil
}

func (a *SSMAdapter) GetParametersByPath(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var req struct {
		Path string `json:"Path"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	var list []*ParameterMeta
	for name, p := range a.params {
		if strings.HasPrefix(name, req.Path) {
			list = append(list, p)
		}
	}

	return map[string]interface{}{"Parameters": list}, nil
}

func (a *SSMAdapter) DeleteParameter(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var req struct {
		Name string `json:"Name"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	delete(a.params, req.Name)
	return map[string]interface{}{}, nil
}

func (a *SSMAdapter) ResourceTypes() []services.ResourceType {
	return []services.ResourceType{
		{Name: "Parameter", ARNFormat: "arn:aws:ssm:{region}:{account}:parameter{name}", Tags: true, Columns: []string{"Name", "Type", "Version", "LastModifiedDate"}},
	}
}

func (a *SSMAdapter) IAMActions() []services.ActionDef {
	return []services.ActionDef{
		{Name: "ssm:GetParameter", ResourceARNs: []string{"arn:aws:ssm:*:*:parameter/*"}},
		{Name: "ssm:PutParameter", ResourceARNs: []string{"arn:aws:ssm:*:*:parameter/*"}},
	}
}

func (a *SSMAdapter) HealthCheck() services.Health {
	return services.Health{Status: "healthy"}
}

func (a *SSMAdapter) EmitEvents() []services.EventDef {
	return []services.EventDef{{Name: "GetParameter", Source: "ssm.amazonaws.com"}}
}

func (a *SSMAdapter) Metrics() []services.MetricDef {
	return []services.MetricDef{}
}

func (a *SSMAdapter) Quotas() []services.Quota {
	return []services.Quota{}
}

func (a *SSMAdapter) Reset(account, region string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.params = make(map[string]*ParameterMeta)
	return nil
}

func (a *SSMAdapter) Snapshot() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (a *SSMAdapter) Restore(data map[string]interface{}) error {
	return nil
}
