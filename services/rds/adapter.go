package rds

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/imrandev/localcloud/gateway/pkg/context"
	"github.com/imrandev/localcloud/services"
)

type RDSAdapter struct {
	mu        sync.RWMutex
	instances map[string]*DBInstance
}

type DBInstance struct {
	DBInstanceIdentifier string `json:"identifier"`
	DBInstanceClass      string `json:"class"`
	Engine               string `json:"engine"`
	DBInstanceStatus     string `json:"status"`
	EndpointAddress      string `json:"endpoint"`
	EndpointPort         int    `json:"port"`
}

var DefaultRDS = &RDSAdapter{
	instances: make(map[string]*DBInstance),
}

func init() {
	services.Register(DefaultRDS)
}

func GetDBInstances() []*DBInstance {
	DefaultRDS.mu.RLock()
	defer DefaultRDS.mu.RUnlock()
	var list []*DBInstance
	for _, inst := range DefaultRDS.instances {
		list = append(list, inst)
	}
	return list
}

func (a *RDSAdapter) Name() string {
	return "rds"
}

func (a *RDSAdapter) Protocol() services.Protocol {
	return services.ProtocolQuery
}

func (a *RDSAdapter) Operations() map[string]services.Handler {
	return map[string]services.Handler{
		"CreateDBInstance":    a.CreateDBInstance,
		"DescribeDBInstances": a.DescribeDBInstances,
		"DeleteDBInstance":    a.DeleteDBInstance,
	}
}

func (a *RDSAdapter) CreateDBInstance(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	id := ctx.ParsedParams["DBInstanceIdentifier"]
	class := ctx.ParsedParams["DBInstanceClass"]
	engine := ctx.ParsedParams["Engine"]
	if engine == "" {
		engine = "postgres"
	}
	if class == "" {
		class = "db.t3.micro"
	}

	inst := &DBInstance{
		DBInstanceIdentifier: id,
		DBInstanceClass:      class,
		Engine:               engine,
		DBInstanceStatus:     "available",
		EndpointAddress:      fmt.Sprintf("%s.localcloud.rds:5432", id),
		EndpointPort:         5432,
	}
	a.instances[id] = inst

	xml := fmt.Sprintf(`    <DBInstance>
      <DBInstanceIdentifier>%s</DBInstanceIdentifier>
      <DBInstanceClass>%s</DBInstanceClass>
      <Engine>%s</Engine>
      <DBInstanceStatus>available</DBInstanceStatus>
      <Endpoint>
        <Address>%s</Address>
        <Port>%d</Port>
      </Endpoint>
    </DBInstance>`, inst.DBInstanceIdentifier, inst.DBInstanceClass, inst.Engine, inst.EndpointAddress, inst.EndpointPort)

	return []byte(xml), nil
}

func (a *RDSAdapter) DescribeDBInstances(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var members string
	for _, inst := range a.instances {
		members += fmt.Sprintf(`      <DBInstance>
        <DBInstanceIdentifier>%s</DBInstanceIdentifier>
        <DBInstanceClass>%s</DBInstanceClass>
        <Engine>%s</Engine>
        <DBInstanceStatus>%s</DBInstanceStatus>
        <Endpoint>
          <Address>%s</Address>
          <Port>%d</Port>
        </Endpoint>
      </DBInstance>
`, inst.DBInstanceIdentifier, inst.DBInstanceClass, inst.Engine, inst.DBInstanceStatus, inst.EndpointAddress, inst.EndpointPort)
	}

	xml := fmt.Sprintf(`    <DBInstances>
%s    </DBInstances>`, members)
	return []byte(xml), nil
}

func (a *RDSAdapter) DeleteDBInstance(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	id := ctx.ParsedParams["DBInstanceIdentifier"]
	delete(a.instances, id)
	return []byte(""), nil
}

func (a *RDSAdapter) ResourceTypes() []services.ResourceType {
	return []services.ResourceType{
		{Name: "DBInstance", ARNFormat: "arn:aws:rds:{region}:{account}:db:{id}", Tags: true, Columns: []string{"DBInstanceIdentifier", "Engine", "DBInstanceStatus", "EndpointAddress"}},
	}
}

func (a *RDSAdapter) IAMActions() []services.ActionDef {
	return []services.ActionDef{{Name: "rds:CreateDBInstance", ResourceARNs: []string{"*"}}}
}

func (a *RDSAdapter) HealthCheck() services.Health {
	return services.Health{Status: "healthy"}
}

func (a *RDSAdapter) EmitEvents() []services.EventDef {
	return []services.EventDef{{Name: "CreateDBInstance", Source: "rds.amazonaws.com"}}
}

func (a *RDSAdapter) Metrics() []services.MetricDef {
	return []services.MetricDef{}
}

func (a *RDSAdapter) Quotas() []services.Quota {
	return []services.Quota{}
}

func (a *RDSAdapter) Reset(account, region string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.instances = make(map[string]*DBInstance)
	return nil
}

func (a *RDSAdapter) Snapshot() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (a *RDSAdapter) Restore(data map[string]interface{}) error {
	return nil
}
