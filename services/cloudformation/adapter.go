package cloudformation

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/imrandev/localcloud/gateway/pkg/context"
	"github.com/imrandev/localcloud/services"
)

type CloudFormationAdapter struct {
	mu     sync.RWMutex
	stacks map[string]*Stack
}

type Stack struct {
	Name        string    `json:"name"`
	Id          string    `json:"id"`
	Status      string    `json:"status"`
	CreatedTime time.Time `json:"created_time"`
}

var DefaultCFN = &CloudFormationAdapter{
	stacks: make(map[string]*Stack),
}

func init() {
	services.Register(DefaultCFN)
}

func GetStacks() []*Stack {
	DefaultCFN.mu.RLock()
	defer DefaultCFN.mu.RUnlock()
	var list []*Stack
	for _, s := range DefaultCFN.stacks {
		list = append(list, s)
	}
	return list
}

func (a *CloudFormationAdapter) Name() string {
	return "cloudformation"
}

func (a *CloudFormationAdapter) Protocol() services.Protocol {
	return services.ProtocolQuery
}

func (a *CloudFormationAdapter) Operations() map[string]services.Handler {
	return map[string]services.Handler{
		"CreateStack":    a.CreateStack,
		"DescribeStacks": a.DescribeStacks,
		"DeleteStack":    a.DeleteStack,
		"ListStacks":     a.ListStacks,
	}
}

func (a *CloudFormationAdapter) CreateStack(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	stackName := ctx.ParsedParams["StackName"]
	if stackName == "" {
		return nil, fmt.Errorf("StackName is required")
	}

	stackId := fmt.Sprintf("arn:aws:cloudformation:%s:%s:stack/%s/%s", ctx.Region, ctx.AccountID, stackName, uuid.New().String())
	a.stacks[stackName] = &Stack{
		Name:        stackName,
		Id:          stackId,
		Status:      "CREATE_COMPLETE",
		CreatedTime: time.Now().UTC(),
	}

	xml := fmt.Sprintf(`    <StackId>%s</StackId>`, stackId)
	return []byte(xml), nil
}

func (a *CloudFormationAdapter) DescribeStacks(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var members string
	for _, s := range a.stacks {
		members += fmt.Sprintf(`      <member>
        <StackName>%s</StackName>
        <StackId>%s</StackId>
        <StackStatus>%s</StackStatus>
        <CreationTime>%s</CreationTime>
      </member>
`, s.Name, s.Id, s.Status, s.CreatedTime.Format(time.RFC3339))
	}

	xml := fmt.Sprintf(`    <Stacks>
%s    </Stacks>`, members)
	return []byte(xml), nil
}

func (a *CloudFormationAdapter) DeleteStack(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	stackName := ctx.ParsedParams["StackName"]
	delete(a.stacks, stackName)
	return []byte(""), nil
}

func (a *CloudFormationAdapter) ListStacks(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	return a.DescribeStacks(ctx, r)
}

func (a *CloudFormationAdapter) ResourceTypes() []services.ResourceType {
	return []services.ResourceType{
		{Name: "Stack", ARNFormat: "arn:aws:cloudformation:{region}:{account}:stack/{name}/{id}", Tags: true, Columns: []string{"StackName", "StackStatus", "CreationTime"}},
	}
}

func (a *CloudFormationAdapter) IAMActions() []services.ActionDef {
	return []services.ActionDef{{Name: "cloudformation:CreateStack", ResourceARNs: []string{"*"}}}
}

func (a *CloudFormationAdapter) HealthCheck() services.Health {
	return services.Health{Status: "healthy"}
}

func (a *CloudFormationAdapter) EmitEvents() []services.EventDef {
	return []services.EventDef{{Name: "CreateStack", Source: "cloudformation.amazonaws.com"}}
}

func (a *CloudFormationAdapter) Metrics() []services.MetricDef {
	return []services.MetricDef{}
}

func (a *CloudFormationAdapter) Quotas() []services.Quota {
	return []services.Quota{}
}

func (a *CloudFormationAdapter) Reset(account, region string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.stacks = make(map[string]*Stack)
	return nil
}

func (a *CloudFormationAdapter) Snapshot() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (a *CloudFormationAdapter) Restore(data map[string]interface{}) error {
	return nil
}
