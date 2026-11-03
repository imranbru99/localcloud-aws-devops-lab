package services

import (
	"net/http"

	"github.com/imrandev/localcloud/gateway/pkg/context"
)

type Protocol string

const (
	ProtocolQuery    Protocol = "query"
	ProtocolJSON10   Protocol = "json10"
	ProtocolJSON11   Protocol = "json11"
	ProtocolRestXML  Protocol = "rest-xml"
	ProtocolRestJSON Protocol = "rest-json"
	ProtocolEC2      Protocol = "ec2"
)

type Handler func(ctx *context.RequestContext, r *http.Request) (interface{}, error)

type ResourceType struct {
	Name      string   `json:"name"`
	ARNFormat string   `json:"arn_format"`
	Tags      bool     `json:"tags"`
	Columns   []string `json:"columns"`
}

type ActionDef struct {
	Name         string   `json:"name"`
	ResourceARNs []string `json:"resource_arns"`
	Conditions   []string `json:"conditions"`
}

type Health struct {
	Status  string            `json:"status"`
	Details map[string]string `json:"details,omitempty"`
}

type EventDef struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}

type MetricDef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Unit      string `json:"unit"`
}

type Quota struct {
	Name         string  `json:"name"`
	DefaultValue float64 `json:"default_value"`
	Description  string  `json:"description"`
}

type ServiceAdapter interface {
	Name() string
	Protocol() Protocol
	Operations() map[string]Handler
	ResourceTypes() []ResourceType
	IAMActions() []ActionDef
	HealthCheck() Health
	EmitEvents() []EventDef
	Metrics() []MetricDef
	Quotas() []Quota
	Reset(account, region string) error
	Snapshot() (map[string]interface{}, error)
	Restore(data map[string]interface{}) error
}
