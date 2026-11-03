package route53

import (
	"encoding/xml"
	
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/imrandev/localcloud/gateway/pkg/context"
	"github.com/imrandev/localcloud/services"
)

type Route53Adapter struct {
	mu    sync.RWMutex
	zones map[string]*HostedZone
}

type HostedZone struct {
	Id          string
	Name        string
	CallerRef   string
	RecordCount int
}

func init() {
	services.Register(&Route53Adapter{
		zones: make(map[string]*HostedZone),
	})
}

func (a *Route53Adapter) Name() string {
	return "route53"
}

func (a *Route53Adapter) Protocol() services.Protocol {
	return services.ProtocolRestXML
}

func (a *Route53Adapter) Operations() map[string]services.Handler {
	return map[string]services.Handler{
		"CreateHostedZone": a.CreateHostedZone,
		"ListHostedZones":  a.ListHostedZones,
	}
}

type CreateHostedZoneResponse struct {
	XMLName    xml.Name   `xml:"CreateHostedZoneResponse"`
	HostedZone ZoneDetail `xml:"HostedZone"`
}

type ListHostedZonesResponse struct {
	XMLName     xml.Name     `xml:"ListHostedZonesResponse"`
	HostedZones []ZoneDetail `xml:"HostedZones>HostedZone"`
}

type ZoneDetail struct {
	Id                     string `xml:"Id"`
	Name                   string `xml:"Name"`
	CallerReference        string `xml:"CallerReference"`
	ResourceRecordSetCount int    `xml:"ResourceRecordSetCount"`
}

func (a *Route53Adapter) CreateHostedZone(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	zoneId := "/hostedzone/Z" + stringsUpperUUID()[:12]
	name := ctx.ParsedParams["Name"]
	if name == "" {
		name = "example.com."
	}

	hz := &HostedZone{
		Id:          zoneId,
		Name:        name,
		CallerRef:   uuid.New().String(),
		RecordCount: 2,
	}
	a.zones[zoneId] = hz

	return CreateHostedZoneResponse{
		HostedZone: ZoneDetail{
			Id:                     hz.Id,
			Name:                   hz.Name,
			CallerReference:        hz.CallerRef,
			ResourceRecordSetCount: hz.RecordCount,
		},
	}, nil
}

func (a *Route53Adapter) ListHostedZones(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var list []ZoneDetail
	for _, z := range a.zones {
		list = append(list, ZoneDetail{
			Id:                     z.Id,
			Name:                   z.Name,
			CallerReference:        z.CallerRef,
			ResourceRecordSetCount: z.RecordCount,
		})
	}

	return ListHostedZonesResponse{
		HostedZones: list,
	}, nil
}

func (a *Route53Adapter) ResourceTypes() []services.ResourceType {
	return []services.ResourceType{
		{Name: "HostedZone", ARNFormat: "arn:aws:route53:::hostedzone/{id}", Tags: true, Columns: []string{"Id", "Name", "RecordCount"}},
	}
}

func (a *Route53Adapter) IAMActions() []services.ActionDef {
	return []services.ActionDef{{Name: "route53:CreateHostedZone", ResourceARNs: []string{"*"}}}
}

func (a *Route53Adapter) HealthCheck() services.Health {
	return services.Health{Status: "healthy"}
}

func (a *Route53Adapter) EmitEvents() []services.EventDef {
	return []services.EventDef{{Name: "CreateHostedZone", Source: "route53.amazonaws.com"}}
}

func (a *Route53Adapter) Metrics() []services.MetricDef {
	return []services.MetricDef{}
}

func (a *Route53Adapter) Quotas() []services.Quota {
	return []services.Quota{}
}

func (a *Route53Adapter) Reset(account, region string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.zones = make(map[string]*HostedZone)
	return nil
}

func (a *Route53Adapter) Snapshot() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (a *Route53Adapter) Restore(data map[string]interface{}) error {
	return nil
}

func stringsUpperUUID() string {
	u := uuid.New().String()
	var out []rune
	for _, ch := range u {
		if ch != '-' {
			if ch >= 'a' && ch <= 'z' {
				ch -= 32
			}
			out = append(out, ch)
		}
	}
	return string(out)
}
