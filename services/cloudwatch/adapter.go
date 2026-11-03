package cloudwatch

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/imrandev/localcloud/gateway/pkg/context"
	"github.com/imrandev/localcloud/services"
)

type CloudWatchAdapter struct {
	mu      sync.RWMutex
	metrics []MetricDatum
	alarms  map[string]*AlarmMeta
}

type MetricDatum struct {
	Namespace  string
	MetricName string
	Value      float64
	Unit       string
}

type AlarmMeta struct {
	AlarmName          string
	AlarmArn           string
	MetricName         string
	Namespace          string
	Threshold          float64
	ComparisonOperator string
	StateValue         string
}

func init() {
	services.Register(&CloudWatchAdapter{
		metrics: make([]MetricDatum, 0),
		alarms:  make(map[string]*AlarmMeta),
	})
}

func (a *CloudWatchAdapter) Name() string {
	return "cloudwatch"
}

func (a *CloudWatchAdapter) Protocol() services.Protocol {
	return services.ProtocolQuery
}

func (a *CloudWatchAdapter) Operations() map[string]services.Handler {
	return map[string]services.Handler{
		"PutMetricData":   a.PutMetricData,
		"ListMetrics":     a.ListMetrics,
		"PutMetricAlarm":  a.PutMetricAlarm,
		"DescribeAlarms":  a.DescribeAlarms,
	}
}

func (a *CloudWatchAdapter) PutMetricData(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	ns := ctx.ParsedParams["Namespace"]
	name := ctx.ParsedParams["MetricData.member.1.MetricName"]

	a.metrics = append(a.metrics, MetricDatum{
		Namespace:  ns,
		MetricName: name,
		Value:      1.0,
		Unit:       "Count",
	})

	return []byte(""), nil
}

func (a *CloudWatchAdapter) ListMetrics(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var members string
	seen := make(map[string]bool)
	for _, m := range a.metrics {
		k := m.Namespace + "/" + m.MetricName
		if !seen[k] {
			seen[k] = true
			members += fmt.Sprintf(`      <member>
        <Namespace>%s</Namespace>
        <MetricName>%s</MetricName>
      </member>
`, m.Namespace, m.MetricName)
		}
	}

	xml := fmt.Sprintf(`    <Metrics>
%s    </Metrics>`, members)
	return []byte(xml), nil
}

func (a *CloudWatchAdapter) PutMetricAlarm(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	alarmName := ctx.ParsedParams["AlarmName"]
	metricName := ctx.ParsedParams["MetricName"]
	ns := ctx.ParsedParams["Namespace"]

	arn := fmt.Sprintf("arn:aws:cloudwatch:%s:%s:alarm:%s", ctx.Region, ctx.AccountID, alarmName)
	a.alarms[alarmName] = &AlarmMeta{
		AlarmName:  alarmName,
		AlarmArn:   arn,
		MetricName: metricName,
		Namespace:  ns,
		StateValue: "OK",
	}

	return []byte(""), nil
}

func (a *CloudWatchAdapter) DescribeAlarms(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var members string
	for _, alarm := range a.alarms {
		members += fmt.Sprintf(`      <member>
        <AlarmName>%s</AlarmName>
        <AlarmArn>%s</AlarmArn>
        <StateValue>%s</StateValue>
        <MetricName>%s</MetricName>
        <Namespace>%s</Namespace>
      </member>
`, alarm.AlarmName, alarm.AlarmArn, alarm.StateValue, alarm.MetricName, alarm.Namespace)
	}

	xml := fmt.Sprintf(`    <MetricAlarms>
%s    </MetricAlarms>`, members)
	return []byte(xml), nil
}

func (a *CloudWatchAdapter) ResourceTypes() []services.ResourceType {
	return []services.ResourceType{
		{Name: "Alarm", ARNFormat: "arn:aws:cloudwatch:{region}:{account}:alarm:{name}", Tags: true, Columns: []string{"AlarmName", "StateValue", "MetricName", "Namespace"}},
	}
}

func (a *CloudWatchAdapter) IAMActions() []services.ActionDef {
	return []services.ActionDef{
		{Name: "cloudwatch:PutMetricData", ResourceARNs: []string{"*"}},
		{Name: "cloudwatch:ListMetrics", ResourceARNs: []string{"*"}},
	}
}

func (a *CloudWatchAdapter) HealthCheck() services.Health {
	return services.Health{Status: "healthy"}
}

func (a *CloudWatchAdapter) EmitEvents() []services.EventDef {
	return []services.EventDef{}
}

func (a *CloudWatchAdapter) Metrics() []services.MetricDef {
	return []services.MetricDef{}
}

func (a *CloudWatchAdapter) Quotas() []services.Quota {
	return []services.Quota{}
}

func (a *CloudWatchAdapter) Reset(account, region string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.metrics = make([]MetricDatum, 0)
	a.alarms = make(map[string]*AlarmMeta)
	return nil
}

func (a *CloudWatchAdapter) Snapshot() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (a *CloudWatchAdapter) Restore(data map[string]interface{}) error {
	return nil
}
