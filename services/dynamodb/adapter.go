package dynamodb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/imrandev/localcloud/gateway/pkg/context"
	"github.com/imrandev/localcloud/services"
)

type DynamoDBAdapter struct {
	mu     sync.RWMutex
	tables map[string]*Table
}

type Table struct {
	TableName            string                   `json:"TableName"`
	TableArn             string                   `json:"TableArn"`
	TableStatus          string                   `json:"TableStatus"`
	CreationDateTime     float64                  `json:"CreationDateTime"`
	ItemCount            int64                    `json:"ItemCount"`
	TableSizeBytes       int64                    `json:"TableSizeBytes"`
	KeySchema            []map[string]string      `json:"KeySchema"`
	AttributeDefinitions []map[string]string      `json:"AttributeDefinitions"`
	Items                []map[string]interface{} `json:"-"`
}

func init() {
	services.Register(&DynamoDBAdapter{
		tables: make(map[string]*Table),
	})
}

func (a *DynamoDBAdapter) Name() string {
	return "dynamodb"
}

func (a *DynamoDBAdapter) Protocol() services.Protocol {
	return services.ProtocolJSON10
}

func (a *DynamoDBAdapter) Operations() map[string]services.Handler {
	return map[string]services.Handler{
		"CreateTable":   a.CreateTable,
		"DescribeTable": a.DescribeTable,
		"ListTables":    a.ListTables,
		"DeleteTable":   a.DeleteTable,
		"PutItem":       a.PutItem,
		"GetItem":       a.GetItem,
		"DeleteItem":    a.DeleteItem,
		"Scan":          a.Scan,
	}
}

func (a *DynamoDBAdapter) CreateTable(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var req struct {
		TableName            string              `json:"TableName"`
		KeySchema            []map[string]string `json:"KeySchema"`
		AttributeDefinitions []map[string]string `json:"AttributeDefinitions"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	if req.TableName == "" {
		return nil, fmt.Errorf("TableName is required")
	}

	tableArn := fmt.Sprintf("arn:aws:dynamodb:%s:%s:table/%s", ctx.Region, ctx.AccountID, req.TableName)
	t := &Table{
		TableName:            req.TableName,
		TableArn:             tableArn,
		TableStatus:          "ACTIVE",
		CreationDateTime:     float64(time.Now().Unix()),
		ItemCount:            0,
		TableSizeBytes:       0,
		KeySchema:            req.KeySchema,
		AttributeDefinitions: req.AttributeDefinitions,
		Items:                make([]map[string]interface{}, 0),
	}

	a.tables[req.TableName] = t
	return map[string]interface{}{"TableDescription": t}, nil
}

func (a *DynamoDBAdapter) DescribeTable(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var req struct {
		TableName string `json:"TableName"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	t, ok := a.tables[req.TableName]
	if !ok {
		return nil, fmt.Errorf("ResourceNotFoundException: Cannot do operations on a non-existent table")
	}

	return map[string]interface{}{"Table": t}, nil
}

func (a *DynamoDBAdapter) ListTables(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	names := make([]string, 0, len(a.tables))
	for name := range a.tables {
		names = append(names, name)
	}

	return map[string]interface{}{"TableNames": names}, nil
}

func (a *DynamoDBAdapter) DeleteTable(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var req struct {
		TableName string `json:"TableName"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	t, ok := a.tables[req.TableName]
	if !ok {
		return nil, fmt.Errorf("ResourceNotFoundException: Cannot do operations on a non-existent table")
	}

	delete(a.tables, req.TableName)
	return map[string]interface{}{"TableDescription": t}, nil
}

func (a *DynamoDBAdapter) PutItem(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var req struct {
		TableName string                 `json:"TableName"`
		Item      map[string]interface{} `json:"Item"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	t, ok := a.tables[req.TableName]
	if !ok {
		return nil, fmt.Errorf("ResourceNotFoundException: Cannot do operations on a non-existent table")
	}

	t.Items = append(t.Items, req.Item)
	t.ItemCount = int64(len(t.Items))

	return map[string]interface{}{}, nil
}

func (a *DynamoDBAdapter) GetItem(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var req struct {
		TableName string                 `json:"TableName"`
		Key       map[string]interface{} `json:"Key"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	t, ok := a.tables[req.TableName]
	if !ok {
		return nil, fmt.Errorf("ResourceNotFoundException")
	}

	for _, item := range t.Items {
		match := true
		for k, v := range req.Key {
			if fmt.Sprintf("%v", item[k]) != fmt.Sprintf("%v", v) {
				match = false
				break
			}
		}
		if match {
			return map[string]interface{}{"Item": item}, nil
		}
	}

	return map[string]interface{}{}, nil
}

func (a *DynamoDBAdapter) DeleteItem(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var req struct {
		TableName string                 `json:"TableName"`
		Key       map[string]interface{} `json:"Key"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	t, ok := a.tables[req.TableName]
	if !ok {
		return nil, fmt.Errorf("ResourceNotFoundException")
	}

	var filtered []map[string]interface{}
	for _, item := range t.Items {
		match := true
		for k, v := range req.Key {
			if fmt.Sprintf("%v", item[k]) != fmt.Sprintf("%v", v) {
				match = false
				break
			}
		}
		if !match {
			filtered = append(filtered, item)
		}
	}
	t.Items = filtered
	t.ItemCount = int64(len(t.Items))

	return map[string]interface{}{}, nil
}

func (a *DynamoDBAdapter) Scan(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var req struct {
		TableName string `json:"TableName"`
	}
	_ = json.Unmarshal(ctx.RawBody, &req)

	t, ok := a.tables[req.TableName]
	if !ok {
		return nil, fmt.Errorf("ResourceNotFoundException")
	}

	return map[string]interface{}{
		"Items": t.Items,
		"Count": len(t.Items),
		"ScannedCount": len(t.Items),
	}, nil
}

func (a *DynamoDBAdapter) ResourceTypes() []services.ResourceType {
	return []services.ResourceType{
		{Name: "Table", ARNFormat: "arn:aws:dynamodb:{region}:{account}:table/{tablename}", Tags: true, Columns: []string{"TableName", "TableStatus", "ItemCount", "CreationDateTime"}},
	}
}

func (a *DynamoDBAdapter) IAMActions() []services.ActionDef {
	return []services.ActionDef{
		{Name: "dynamodb:CreateTable", ResourceARNs: []string{"arn:aws:dynamodb:*:*:table/*"}},
		{Name: "dynamodb:PutItem", ResourceARNs: []string{"arn:aws:dynamodb:*:*:table/*"}},
		{Name: "dynamodb:GetItem", ResourceARNs: []string{"arn:aws:dynamodb:*:*:table/*"}},
		{Name: "dynamodb:Scan", ResourceARNs: []string{"arn:aws:dynamodb:*:*:table/*"}},
	}
}

func (a *DynamoDBAdapter) HealthCheck() services.Health {
	return services.Health{Status: "healthy"}
}

func (a *DynamoDBAdapter) EmitEvents() []services.EventDef {
	return []services.EventDef{
		{Name: "CreateTable", Source: "dynamodb.amazonaws.com"},
		{Name: "PutItem", Source: "dynamodb.amazonaws.com"},
	}
}

func (a *DynamoDBAdapter) Metrics() []services.MetricDef {
	return []services.MetricDef{}
}

func (a *DynamoDBAdapter) Quotas() []services.Quota {
	return []services.Quota{}
}

func (a *DynamoDBAdapter) Reset(account, region string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tables = make(map[string]*Table)
	return nil
}

func (a *DynamoDBAdapter) Snapshot() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (a *DynamoDBAdapter) Restore(data map[string]interface{}) error {
	return nil
}
