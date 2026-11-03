package sts

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/imrandev/localcloud/gateway/pkg/context"
	"github.com/imrandev/localcloud/services"
)

type STSAdapter struct{}

func init() {
	services.Register(&STSAdapter{})
}

func (a *STSAdapter) Name() string {
	return "sts"
}

func (a *STSAdapter) Protocol() services.Protocol {
	return services.ProtocolQuery
}

func (a *STSAdapter) Operations() map[string]services.Handler {
	return map[string]services.Handler{
		"GetCallerIdentity": a.GetCallerIdentity,
		"AssumeRole":        a.AssumeRole,
		"GetSessionToken":   a.GetSessionToken,
	}
}

func (a *STSAdapter) GetCallerIdentity(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	userId := ctx.AccessKeyID
	if userId == "" {
		userId = "AKIAIOSFODNN7EXAMPLE"
	}
	arn := ctx.PrincipalARN
	if arn == "" {
		arn = fmt.Sprintf("arn:aws:iam::%s:root", ctx.AccountID)
	}

	xml := fmt.Sprintf(`    <Arn>%s</Arn>
    <UserId>%s</UserId>
    <Account>%s</Account>`, arn, userId, ctx.AccountID)

	return []byte(xml), nil
}

func (a *STSAdapter) AssumeRole(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	roleArn := ctx.ParsedParams["RoleArn"]
	roleSessionName := ctx.ParsedParams["RoleSessionName"]
	if roleSessionName == "" {
		roleSessionName = "localcloud-session"
	}

	accessKey := "ASIA" + stringsUpperUUID()
	secretKey := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	sessionToken := "FwoGZXIvYXdzEAEaE..." + uuid.New().String()
	exp := time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339)

	assumedRoleArn := roleArn
	if assumedRoleArn == "" {
		assumedRoleArn = fmt.Sprintf("arn:aws:sts::%s:assumed-role/LearnerRole/%s", ctx.AccountID, roleSessionName)
	}

	xml := fmt.Sprintf(`    <Credentials>
      <AccessKeyId>%s</AccessKeyId>
      <SecretAccessKey>%s</SecretAccessKey>
      <SessionToken>%s</SessionToken>
      <Expiration>%s</Expiration>
    </Credentials>
    <AssumedRoleUser>
      <AssumedRoleId>%s:%s</AssumedRoleId>
      <Arn>%s</Arn>
    </AssumedRoleUser>`, accessKey, secretKey, sessionToken, exp, accessKey, roleSessionName, assumedRoleArn)

	return []byte(xml), nil
}

func (a *STSAdapter) GetSessionToken(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	accessKey := "ASIA" + stringsUpperUUID()
	secretKey := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	sessionToken := "FwoGZXIvYXdzEAEaE..." + uuid.New().String()
	exp := time.Now().UTC().Add(12 * time.Hour).Format(time.RFC3339)

	xml := fmt.Sprintf(`    <Credentials>
      <AccessKeyId>%s</AccessKeyId>
      <SecretAccessKey>%s</SecretAccessKey>
      <SessionToken>%s</SessionToken>
      <Expiration>%s</Expiration>
    </Credentials>`, accessKey, secretKey, sessionToken, exp)

	return []byte(xml), nil
}

func (a *STSAdapter) ResourceTypes() []services.ResourceType {
	return []services.ResourceType{}
}

func (a *STSAdapter) IAMActions() []services.ActionDef {
	return []services.ActionDef{
		{Name: "sts:GetCallerIdentity", ResourceARNs: []string{"*"}},
		{Name: "sts:AssumeRole", ResourceARNs: []string{"arn:aws:iam::*:role/*"}},
		{Name: "sts:GetSessionToken", ResourceARNs: []string{"*"}},
	}
}

func (a *STSAdapter) HealthCheck() services.Health {
	return services.Health{Status: "healthy"}
}

func (a *STSAdapter) EmitEvents() []services.EventDef {
	return []services.EventDef{{Name: "GetCallerIdentity", Source: "sts.amazonaws.com"}}
}

func (a *STSAdapter) Metrics() []services.MetricDef {
	return []services.MetricDef{}
}

func (a *STSAdapter) Quotas() []services.Quota {
	return []services.Quota{}
}

func (a *STSAdapter) Reset(account, region string) error {
	return nil
}

func (a *STSAdapter) Snapshot() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (a *STSAdapter) Restore(data map[string]interface{}) error {
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
	if len(out) > 16 {
		return string(out[:16])
	}
	return string(out)
}
