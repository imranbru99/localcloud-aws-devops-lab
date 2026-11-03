package secretsmanager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/imrandev/localcloud/gateway/pkg/context"
	"github.com/imrandev/localcloud/services"
)

type SecretsManagerAdapter struct {
	mu      sync.RWMutex
	secrets map[string]*SecretMeta
}

type SecretMeta struct {
	ARN          string `json:"ARN"`
	Name         string `json:"Name"`
	Description  string `json:"Description"`
	SecretString string `json:"SecretString"`
	CreatedDate  int64  `json:"CreatedDate"`
}

func init() {
	services.Register(&SecretsManagerAdapter{
		secrets: make(map[string]*SecretMeta),
	})
}

func (a *SecretsManagerAdapter) Name() string {
	return "secretsmanager"
}

func (a *SecretsManagerAdapter) Protocol() services.Protocol {
	return services.ProtocolJSON11
}

func (a *SecretsManagerAdapter) Operations() map[string]services.Handler {
	return map[string]services.Handler{
		"CreateSecret":    a.CreateSecret,
		"GetSecretValue":  a.GetSecretValue,
		"DescribeSecret":  a.DescribeSecret,
		"ListSecrets":     a.ListSecrets,
		"PutSecretValue":  a.PutSecretValue,
		"DeleteSecret":    a.DeleteSecret,
	}
}

func (a *SecretsManagerAdapter) CreateSecret(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var req struct {
		Name         string `json:"Name"`
		Description  string `json:"Description"`
		SecretString string `json:"SecretString"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	if req.Name == "" {
		return nil, fmt.Errorf("Name is required")
	}

	arn := fmt.Sprintf("arn:aws:secretsmanager:%s:%s:secret:%s-%s", ctx.Region, ctx.AccountID, req.Name, uuid.New().String()[:6])
	s := &SecretMeta{
		ARN:          arn,
		Name:         req.Name,
		Description:  req.Description,
		SecretString: req.SecretString,
		CreatedDate:  time.Now().Unix(),
	}

	a.secrets[req.Name] = s
	return map[string]interface{}{"ARN": arn, "Name": req.Name}, nil
}

func (a *SecretsManagerAdapter) GetSecretValue(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var req struct {
		SecretId string `json:"SecretId"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	s, ok := a.secrets[req.SecretId]
	if !ok {
		// Try search by ARN or prefix
		for _, sec := range a.secrets {
			if sec.ARN == req.SecretId {
				s = sec
				break
			}
		}
	}

	if s == nil {
		return nil, fmt.Errorf("ResourceNotFoundException: Secrets Manager can't find the specified secret.")
	}

	return map[string]interface{}{
		"ARN":          s.ARN,
		"Name":         s.Name,
		"SecretString": s.SecretString,
		"CreatedDate":  s.CreatedDate,
	}, nil
}

func (a *SecretsManagerAdapter) DescribeSecret(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var req struct {
		SecretId string `json:"SecretId"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	s, ok := a.secrets[req.SecretId]
	if !ok {
		return nil, fmt.Errorf("ResourceNotFoundException")
	}

	return map[string]interface{}{
		"ARN":         s.ARN,
		"Name":        s.Name,
		"Description": s.Description,
	}, nil
}

func (a *SecretsManagerAdapter) ListSecrets(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var list []map[string]interface{}
	for _, s := range a.secrets {
		list = append(list, map[string]interface{}{
			"ARN":         s.ARN,
			"Name":        s.Name,
			"Description": s.Description,
		})
	}

	return map[string]interface{}{"SecretList": list}, nil
}

func (a *SecretsManagerAdapter) PutSecretValue(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var req struct {
		SecretId     string `json:"SecretId"`
		SecretString string `json:"SecretString"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	s, ok := a.secrets[req.SecretId]
	if !ok {
		return nil, fmt.Errorf("ResourceNotFoundException")
	}

	s.SecretString = req.SecretString
	return map[string]interface{}{"ARN": s.ARN, "Name": s.Name}, nil
}

func (a *SecretsManagerAdapter) DeleteSecret(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var req struct {
		SecretId string `json:"SecretId"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	delete(a.secrets, req.SecretId)
	return map[string]interface{}{"Name": req.SecretId}, nil
}

func (a *SecretsManagerAdapter) ResourceTypes() []services.ResourceType {
	return []services.ResourceType{
		{Name: "Secret", ARNFormat: "arn:aws:secretsmanager:{region}:{account}:secret:{name}", Tags: true, Columns: []string{"Name", "ARN", "Description"}},
	}
}

func (a *SecretsManagerAdapter) IAMActions() []services.ActionDef {
	return []services.ActionDef{
		{Name: "secretsmanager:GetSecretValue", ResourceARNs: []string{"arn:aws:secretsmanager:*:*:secret:*"}},
		{Name: "secretsmanager:CreateSecret", ResourceARNs: []string{"arn:aws:secretsmanager:*:*:secret:*"}},
	}
}

func (a *SecretsManagerAdapter) HealthCheck() services.Health {
	return services.Health{Status: "healthy"}
}

func (a *SecretsManagerAdapter) EmitEvents() []services.EventDef {
	return []services.EventDef{
		{Name: "GetSecretValue", Source: "secretsmanager.amazonaws.com"},
	}
}

func (a *SecretsManagerAdapter) Metrics() []services.MetricDef {
	return []services.MetricDef{}
}

func (a *SecretsManagerAdapter) Quotas() []services.Quota {
	return []services.Quota{}
}

func (a *SecretsManagerAdapter) Reset(account, region string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.secrets = make(map[string]*SecretMeta)
	return nil
}

func (a *SecretsManagerAdapter) Snapshot() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (a *SecretsManagerAdapter) Restore(data map[string]interface{}) error {
	return nil
}
