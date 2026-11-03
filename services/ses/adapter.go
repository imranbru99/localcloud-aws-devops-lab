package ses

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/imrandev/localcloud/gateway/pkg/context"
	"github.com/imrandev/localcloud/services"
)

type MailMessage struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Timestamp time.Time `json:"timestamp"`
}

type SESAdapter struct {
	mu         sync.RWMutex
	identities []string
	mailbox    []*MailMessage
}

var DefaultMailbox = &SESAdapter{
	identities: []string{"admin@example.com", "notifications@company.com"},
	mailbox:    make([]*MailMessage, 0),
}

func init() {
	services.Register(DefaultMailbox)
}

func (a *SESAdapter) Name() string {
	return "ses"
}

func (a *SESAdapter) Protocol() services.Protocol {
	return services.ProtocolQuery
}

func (a *SESAdapter) Operations() map[string]services.Handler {
	return map[string]services.Handler{
		"SendEmail":           a.SendEmail,
		"SendRawEmail":        a.SendEmail,
		"VerifyEmailIdentity": a.VerifyEmailIdentity,
		"ListIdentities":      a.ListIdentities,
	}
}

func (a *SESAdapter) SendEmail(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	from := ctx.ParsedParams["Source"]
	to := ctx.ParsedParams["Destination.ToAddresses.member.1"]
	subject := ctx.ParsedParams["Message.Subject.Data"]
	body := ctx.ParsedParams["Message.Body.Text.Data"]
	if body == "" {
		body = ctx.ParsedParams["Message.Body.Html.Data"]
	}

	msgID := uuid.New().String()
	mail := &MailMessage{
		ID:        msgID,
		From:      from,
		To:        to,
		Subject:   subject,
		Body:      body,
		Timestamp: time.Now().UTC(),
	}
	a.mailbox = append(a.mailbox, mail)

	xml := fmt.Sprintf(`    <MessageId>%s</MessageId>`, msgID)
	return []byte(xml), nil
}

func (a *SESAdapter) VerifyEmailIdentity(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	email := ctx.ParsedParams["EmailAddress"]
	if email != "" {
		a.identities = append(a.identities, email)
	}
	return []byte(""), nil
}

func (a *SESAdapter) ListIdentities(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var members string
	for _, id := range a.identities {
		members += fmt.Sprintf(`      <member>%s</member>
`, id)
	}

	xml := fmt.Sprintf(`    <Identities>
%s    </Identities>`, members)
	return []byte(xml), nil
}

func GetMailbox() []*MailMessage {
	DefaultMailbox.mu.RLock()
	defer DefaultMailbox.mu.RUnlock()
	out := make([]*MailMessage, len(DefaultMailbox.mailbox))
	copy(out, DefaultMailbox.mailbox)
	return out
}

func (a *SESAdapter) ResourceTypes() []services.ResourceType {
	return []services.ResourceType{
		{Name: "Identity", ARNFormat: "arn:aws:ses:{region}:{account}:identity/{email}", Tags: false, Columns: []string{"Identity"}},
	}
}

func (a *SESAdapter) IAMActions() []services.ActionDef {
	return []services.ActionDef{
		{Name: "ses:SendEmail", ResourceARNs: []string{"*"}},
	}
}

func (a *SESAdapter) HealthCheck() services.Health {
	return services.Health{Status: "healthy"}
}

func (a *SESAdapter) EmitEvents() []services.EventDef {
	return []services.EventDef{{Name: "SendEmail", Source: "ses.amazonaws.com"}}
}

func (a *SESAdapter) Metrics() []services.MetricDef {
	return []services.MetricDef{}
}

func (a *SESAdapter) Quotas() []services.Quota {
	return []services.Quota{}
}

func (a *SESAdapter) Reset(account, region string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.mailbox = make([]*MailMessage, 0)
	return nil
}

func (a *SESAdapter) Snapshot() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (a *SESAdapter) Restore(data map[string]interface{}) error {
	return nil
}
