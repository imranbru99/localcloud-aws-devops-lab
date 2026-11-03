package lambda

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

type LambdaAdapter struct {
	mu        sync.RWMutex
	functions map[string]*FunctionMeta
}

type FunctionMeta struct {
	FunctionName string            `json:"FunctionName"`
	FunctionArn  string            `json:"FunctionArn"`
	Runtime      string            `json:"Runtime"`
	Role         string            `json:"Role"`
	Handler      string            `json:"Handler"`
	CodeSize     int64             `json:"CodeSize"`
	Description  string            `json:"Description"`
	Timeout      int               `json:"Timeout"`
	MemorySize   int               `json:"MemorySize"`
	LastModified string            `json:"LastModified"`
	Version      string            `json:"Version"`
	Environment  map[string]string `json:"Environment,omitempty"`
}

func init() {
	services.Register(&LambdaAdapter{
		functions: make(map[string]*FunctionMeta),
	})
}

func (a *LambdaAdapter) Name() string {
	return "lambda"
}

func (a *LambdaAdapter) Protocol() services.Protocol {
	return services.ProtocolRestJSON
}

func (a *LambdaAdapter) Operations() map[string]services.Handler {
	return map[string]services.Handler{
		"CreateFunction": a.CreateFunction,
		"GetFunction":    a.GetFunction,
		"ListFunctions":  a.ListFunctions,
		"DeleteFunction": a.DeleteFunction,
		"Invoke":         a.Invoke,
	}
}

func (a *LambdaAdapter) CreateFunction(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var req struct {
		FunctionName string `json:"FunctionName"`
		Runtime      string `json:"Runtime"`
		Role         string `json:"Role"`
		Handler      string `json:"Handler"`
		Description  string `json:"Description"`
		Timeout      int    `json:"Timeout"`
		MemorySize   int    `json:"MemorySize"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	if req.FunctionName == "" {
		return nil, fmt.Errorf("FunctionName is required")
	}
	if req.Runtime == "" {
		req.Runtime = "python3.12"
	}
	if req.Timeout == 0 {
		req.Timeout = 3
	}
	if req.MemorySize == 0 {
		req.MemorySize = 128
	}

	arn := fmt.Sprintf("arn:aws:lambda:%s:%s:function:%s", ctx.Region, ctx.AccountID, req.FunctionName)
	fn := &FunctionMeta{
		FunctionName: req.FunctionName,
		FunctionArn:  arn,
		Runtime:      req.Runtime,
		Role:         req.Role,
		Handler:      req.Handler,
		CodeSize:     1024,
		Description:  req.Description,
		Timeout:      req.Timeout,
		MemorySize:   req.MemorySize,
		LastModified: time.Now().UTC().Format(time.RFC3339),
		Version:      "$LATEST",
	}

	a.functions[req.FunctionName] = fn
	return fn, nil
}

func (a *LambdaAdapter) GetFunction(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	fnName := ctx.ParsedParams["FunctionName"]
	fn, ok := a.functions[fnName]
	if !ok {
		return nil, fmt.Errorf("ResourceNotFoundException: Function not found: %s", fnName)
	}

	return map[string]interface{}{
		"Configuration": fn,
		"Code": map[string]string{
			"Location": "http://localhost:4566/mock-lambda-code.zip",
		},
	}, nil
}

func (a *LambdaAdapter) ListFunctions(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	fns := make([]*FunctionMeta, 0, len(a.functions))
	for _, fn := range a.functions {
		fns = append(fns, fn)
	}

	return map[string]interface{}{
		"Functions": fns,
	}, nil
}

func (a *LambdaAdapter) DeleteFunction(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	fnName := ctx.ParsedParams["FunctionName"]
	delete(a.functions, fnName)
	return nil, nil
}

func (a *LambdaAdapter) Invoke(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	fnName := ctx.ParsedParams["FunctionName"]
	fn, ok := a.functions[fnName]
	if !ok {
		return nil, fmt.Errorf("ResourceNotFoundException: Function not found: %s", fnName)
	}

	// Simulated invocation response with educational execution metrics
	execId := uuid.New().String()
	res := map[string]interface{}{
		"statusCode": 200,
		"body": fmt.Sprintf("Hello from LocalCloud Lambda function '%s' (runtime %s)! Execution ID: %s", fn.FunctionName, fn.Runtime, execId),
	}

	return res, nil
}

func (a *LambdaAdapter) ResourceTypes() []services.ResourceType {
	return []services.ResourceType{
		{Name: "Function", ARNFormat: "arn:aws:lambda:{region}:{account}:function:{functionname}", Tags: true, Columns: []string{"FunctionName", "Runtime", "Handler", "Timeout", "LastModified"}},
	}
}

func (a *LambdaAdapter) IAMActions() []services.ActionDef {
	return []services.ActionDef{
		{Name: "lambda:CreateFunction", ResourceARNs: []string{"arn:aws:lambda:*:*:function:*"}},
		{Name: "lambda:InvokeFunction", ResourceARNs: []string{"arn:aws:lambda:*:*:function:*"}},
	}
}

func (a *LambdaAdapter) HealthCheck() services.Health {
	return services.Health{Status: "healthy"}
}

func (a *LambdaAdapter) EmitEvents() []services.EventDef {
	return []services.EventDef{
		{Name: "Invoke", Source: "lambda.amazonaws.com"},
	}
}

func (a *LambdaAdapter) Metrics() []services.MetricDef {
	return []services.MetricDef{
		{Name: "Invocations", Namespace: "AWS/Lambda", Unit: "Count"},
		{Name: "Duration", Namespace: "AWS/Lambda", Unit: "Milliseconds"},
	}
}

func (a *LambdaAdapter) Quotas() []services.Quota {
	return []services.Quota{}
}

func (a *LambdaAdapter) Reset(account, region string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.functions = make(map[string]*FunctionMeta)
	return nil
}

func (a *LambdaAdapter) Snapshot() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (a *LambdaAdapter) Restore(data map[string]interface{}) error {
	return nil
}
