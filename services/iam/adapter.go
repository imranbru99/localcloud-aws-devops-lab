package iam

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/imrandev/localcloud/gateway/pkg/context"
	gatewayIAM "github.com/imrandev/localcloud/gateway/pkg/iam"
	"github.com/imrandev/localcloud/services"
)

type IAMAdapter struct {
	mu sync.RWMutex
}

func init() {
	services.Register(&IAMAdapter{})
}

func (a *IAMAdapter) Name() string {
	return "iam"
}

func (a *IAMAdapter) Protocol() services.Protocol {
	return services.ProtocolQuery
}

func (a *IAMAdapter) Operations() map[string]services.Handler {
	return map[string]services.Handler{
		"CreateUser":              a.CreateUser,
		"GetUser":                 a.GetUser,
		"ListUsers":               a.ListUsers,
		"DeleteUser":              a.DeleteUser,
		"CreateAccessKey":         a.CreateAccessKey,
		"ListAccessKeys":          a.ListAccessKeys,
		"DeleteAccessKey":         a.DeleteAccessKey,
		"CreateRole":              a.CreateRole,
		"GetRole":                 a.GetRole,
		"ListRoles":               a.ListRoles,
		"DeleteRole":              a.DeleteRole,
		"CreatePolicy":            a.CreatePolicy,
		"GetPolicy":               a.GetPolicy,
		"ListPolicies":            a.ListPolicies,
		"AttachUserPolicy":        a.AttachUserPolicy,
		"DetachUserPolicy":        a.DetachUserPolicy,
		"AttachRolePolicy":        a.AttachRolePolicy,
		"DetachRolePolicy":        a.DetachRolePolicy,
		"SimulateCustomPolicy":    a.SimulateCustomPolicy,
		"SimulatePrincipalPolicy": a.SimulatePrincipalPolicy,
	}
}

func (a *IAMAdapter) CreateUser(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	userName := ctx.ParsedParams["UserName"]
	if userName == "" {
		return nil, fmt.Errorf("UserName parameter is required")
	}

	userId := "AIDA" + stringsUpperUUID()
	userArn := fmt.Sprintf("arn:aws:iam::%s:user/%s", ctx.AccountID, userName)

	gatewayIAM.GlobalIAM.Users[userName] = &gatewayIAM.User{
		UserName: userName,
		Arn:      userArn,
		UserId:   userId,
	}

	xml := fmt.Sprintf(`    <User>
      <Path>/</Path>
      <UserName>%s</UserName>
      <UserId>%s</UserId>
      <Arn>%s</Arn>
      <CreateDate>%s</CreateDate>
    </User>`, userName, userId, userArn, time.Now().UTC().Format(time.RFC3339))

	return []byte(xml), nil
}

func (a *IAMAdapter) GetUser(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	userName := ctx.ParsedParams["UserName"]
	if userName == "" {
		userName = "root"
	}

	user, exists := gatewayIAM.GlobalIAM.Users[userName]
	if !exists {
		// Mock root user if queried
		user = &gatewayIAM.User{
			UserName: userName,
			Arn:      fmt.Sprintf("arn:aws:iam::%s:%s", ctx.AccountID, userName),
			UserId:   "AIDA" + stringsUpperUUID(),
		}
	}

	xml := fmt.Sprintf(`    <User>
      <Path>/</Path>
      <UserName>%s</UserName>
      <UserId>%s</UserId>
      <Arn>%s</Arn>
      <CreateDate>%s</CreateDate>
    </User>`, user.UserName, user.UserId, user.Arn, time.Now().UTC().Format(time.RFC3339))

	return []byte(xml), nil
}

func (a *IAMAdapter) ListUsers(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var members string
	for _, u := range gatewayIAM.GlobalIAM.Users {
		members += fmt.Sprintf(`      <member>
        <Path>/</Path>
        <UserName>%s</UserName>
        <UserId>%s</UserId>
        <Arn>%s</Arn>
        <CreateDate>%s</CreateDate>
      </member>
`, u.UserName, u.UserId, u.Arn, time.Now().UTC().Format(time.RFC3339))
	}

	xml := fmt.Sprintf(`    <Users>
%s    </Users>
    <IsTruncated>false</IsTruncated>`, members)

	return []byte(xml), nil
}

func (a *IAMAdapter) DeleteUser(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	userName := ctx.ParsedParams["UserName"]
	delete(gatewayIAM.GlobalIAM.Users, userName)
	return []byte(""), nil
}

func (a *IAMAdapter) CreateAccessKey(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	userName := ctx.ParsedParams["UserName"]
	if userName == "" {
		userName = "root"
	}
	accessKey := "AKIA" + stringsUpperUUID()
	secretKey := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

	xml := fmt.Sprintf(`    <AccessKey>
      <UserName>%s</UserName>
      <AccessKeyId>%s</AccessKeyId>
      <Status>Active</Status>
      <SecretAccessKey>%s</SecretAccessKey>
      <CreateDate>%s</CreateDate>
    </AccessKey>`, userName, accessKey, secretKey, time.Now().UTC().Format(time.RFC3339))

	return []byte(xml), nil
}

func (a *IAMAdapter) ListAccessKeys(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	userName := ctx.ParsedParams["UserName"]
	xml := fmt.Sprintf(`    <AccessKeyMetadata>
      <member>
        <UserName>%s</UserName>
        <AccessKeyId>%s</AccessKeyId>
        <Status>Active</Status>
        <CreateDate>%s</CreateDate>
      </member>
    </AccessKeyMetadata>
    <IsTruncated>false</IsTruncated>`, userName, ctx.AccessKeyID, time.Now().UTC().Format(time.RFC3339))

	return []byte(xml), nil
}

func (a *IAMAdapter) DeleteAccessKey(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	return []byte(""), nil
}

func (a *IAMAdapter) CreateRole(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	roleName := ctx.ParsedParams["RoleName"]
	assumeRoleDoc := ctx.ParsedParams["AssumeRolePolicyDocument"]
	roleId := "AROA" + stringsUpperUUID()
	roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", ctx.AccountID, roleName)

	gatewayIAM.GlobalIAM.Roles[roleName] = &gatewayIAM.Role{
		RoleName: roleName,
		Arn:      roleArn,
		RoleId:   roleId,
	}

	escapedPolicy := url.QueryEscape(assumeRoleDoc)
	xml := fmt.Sprintf(`    <Role>
      <Path>/</Path>
      <RoleName>%s</RoleName>
      <RoleId>%s</RoleId>
      <Arn>%s</Arn>
      <CreateDate>%s</CreateDate>
      <AssumeRolePolicyDocument>%s</AssumeRolePolicyDocument>
    </Role>`, roleName, roleId, roleArn, time.Now().UTC().Format(time.RFC3339), escapedPolicy)

	return []byte(xml), nil
}

func (a *IAMAdapter) GetRole(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	roleName := ctx.ParsedParams["RoleName"]
	role, exists := gatewayIAM.GlobalIAM.Roles[roleName]
	if !exists {
		role = &gatewayIAM.Role{
			RoleName: roleName,
			Arn:      fmt.Sprintf("arn:aws:iam::%s:role/%s", ctx.AccountID, roleName),
			RoleId:   "AROA" + stringsUpperUUID(),
		}
	}

	xml := fmt.Sprintf(`    <Role>
      <Path>/</Path>
      <RoleName>%s</RoleName>
      <RoleId>%s</RoleId>
      <Arn>%s</Arn>
      <CreateDate>%s</CreateDate>
    </Role>`, role.RoleName, role.RoleId, role.Arn, time.Now().UTC().Format(time.RFC3339))

	return []byte(xml), nil
}

func (a *IAMAdapter) ListRoles(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var members string
	for _, role := range gatewayIAM.GlobalIAM.Roles {
		members += fmt.Sprintf(`      <member>
        <Path>/</Path>
        <RoleName>%s</RoleName>
        <RoleId>%s</RoleId>
        <Arn>%s</Arn>
        <CreateDate>%s</CreateDate>
      </member>
`, role.RoleName, role.RoleId, role.Arn, time.Now().UTC().Format(time.RFC3339))
	}

	xml := fmt.Sprintf(`    <Roles>
%s    </Roles>
    <IsTruncated>false</IsTruncated>`, members)

	return []byte(xml), nil
}

func (a *IAMAdapter) DeleteRole(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	roleName := ctx.ParsedParams["RoleName"]
	delete(gatewayIAM.GlobalIAM.Roles, roleName)
	return []byte(""), nil
}

func (a *IAMAdapter) CreatePolicy(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	policyName := ctx.ParsedParams["PolicyName"]
	policyDocRaw := ctx.ParsedParams["PolicyDocument"]
	policyArn := fmt.Sprintf("arn:aws:iam::%s:policy/%s", ctx.AccountID, policyName)
	policyId := "ANPA" + stringsUpperUUID()

	doc, err := gatewayIAM.ParsePolicyDocument(policyDocRaw)
	if err == nil {
		gatewayIAM.GlobalIAM.Policies[policyArn] = doc
	}

	xml := fmt.Sprintf(`    <Policy>
      <PolicyName>%s</PolicyName>
      <PolicyId>%s</PolicyId>
      <Arn>%s</Arn>
      <Path>/</Path>
      <DefaultVersionId>v1</DefaultVersionId>
      <AttachmentCount>0</AttachmentCount>
      <CreateDate>%s</CreateDate>
      <UpdateDate>%s</UpdateDate>
    </Policy>`, policyName, policyId, policyArn, time.Now().UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339))

	return []byte(xml), nil
}

func (a *IAMAdapter) GetPolicy(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	policyArn := ctx.ParsedParams["PolicyArn"]
	xml := fmt.Sprintf(`    <Policy>
      <Arn>%s</Arn>
      <DefaultVersionId>v1</DefaultVersionId>
      <AttachmentCount>1</AttachmentCount>
    </Policy>`, policyArn)
	return []byte(xml), nil
}

func (a *IAMAdapter) ListPolicies(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var members string
	for arn := range gatewayIAM.GlobalIAM.Policies {
		members += fmt.Sprintf(`      <member>
        <PolicyName>%s</PolicyName>
        <Arn>%s</Arn>
        <DefaultVersionId>v1</DefaultVersionId>
        <AttachmentCount>1</AttachmentCount>
      </member>
`, arn, arn)
	}

	xml := fmt.Sprintf(`    <Policies>
%s    </Policies>
    <IsTruncated>false</IsTruncated>`, members)

	return []byte(xml), nil
}

func (a *IAMAdapter) AttachUserPolicy(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	userName := ctx.ParsedParams["UserName"]
	policyArn := ctx.ParsedParams["PolicyArn"]

	gatewayIAM.GlobalIAM.AttachedUser[userName] = append(gatewayIAM.GlobalIAM.AttachedUser[userName], policyArn)
	return []byte(""), nil
}

func (a *IAMAdapter) DetachUserPolicy(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	userName := ctx.ParsedParams["UserName"]
	policyArn := ctx.ParsedParams["PolicyArn"]

	list := gatewayIAM.GlobalIAM.AttachedUser[userName]
	var updated []string
	for _, p := range list {
		if p != policyArn {
			updated = append(updated, p)
		}
	}
	gatewayIAM.GlobalIAM.AttachedUser[userName] = updated
	return []byte(""), nil
}

func (a *IAMAdapter) AttachRolePolicy(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	roleName := ctx.ParsedParams["RoleName"]
	policyArn := ctx.ParsedParams["PolicyArn"]

	gatewayIAM.GlobalIAM.AttachedRole[roleName] = append(gatewayIAM.GlobalIAM.AttachedRole[roleName], policyArn)
	return []byte(""), nil
}

func (a *IAMAdapter) DetachRolePolicy(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	roleName := ctx.ParsedParams["RoleName"]
	policyArn := ctx.ParsedParams["PolicyArn"]

	list := gatewayIAM.GlobalIAM.AttachedRole[roleName]
	var updated []string
	for _, p := range list {
		if p != policyArn {
			updated = append(updated, p)
		}
	}
	gatewayIAM.GlobalIAM.AttachedRole[roleName] = updated
	return []byte(""), nil
}

func (a *IAMAdapter) SimulateCustomPolicy(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	action := ctx.ParsedParams["ActionNames.member.1"]
	if action == "" {
		action = "s3:ListBucket"
	}
	resArn := ctx.ParsedParams["ResourceArns.member.1"]
	if resArn == "" {
		resArn = "*"
	}

	eval := gatewayIAM.Evaluate(ctx, action, resArn)
	evalDecision := "allowed"
	if !eval.Allowed {
		evalDecision = "implicitDeny"
	}

	xml := fmt.Sprintf(`    <EvaluationResults>
      <member>
        <EvalActionName>%s</EvalActionName>
        <EvalResourceName>%s</EvalResourceName>
        <EvalDecision>%s</EvalDecision>
        <MatchedStatementsDetails/>
      </member>
    </EvaluationResults>
    <IsTruncated>false</IsTruncated>`, action, resArn, evalDecision)

	return []byte(xml), nil
}

func (a *IAMAdapter) SimulatePrincipalPolicy(ctx *context.RequestContext, r *http.Request) (interface{}, error) {
	return a.SimulateCustomPolicy(ctx, r)
}

func (a *IAMAdapter) ResourceTypes() []services.ResourceType {
	return []services.ResourceType{
		{Name: "User", ARNFormat: "arn:aws:iam::{account}:user/{username}", Tags: true, Columns: []string{"UserName", "UserId", "Arn", "CreateDate"}},
		{Name: "Role", ARNFormat: "arn:aws:iam::{account}:role/{rolename}", Tags: true, Columns: []string{"RoleName", "RoleId", "Arn", "CreateDate"}},
		{Name: "Policy", ARNFormat: "arn:aws:iam::{account}:policy/{policyname}", Tags: true, Columns: []string{"PolicyName", "Arn"}},
	}
}

func (a *IAMAdapter) IAMActions() []services.ActionDef {
	return []services.ActionDef{
		{Name: "iam:CreateUser", ResourceARNs: []string{"arn:aws:iam::*:user/*"}},
		{Name: "iam:GetUser", ResourceARNs: []string{"arn:aws:iam::*:user/*"}},
		{Name: "iam:ListUsers", ResourceARNs: []string{"arn:aws:iam::*:user/"}},
		{Name: "iam:DeleteUser", ResourceARNs: []string{"arn:aws:iam::*:user/*"}},
		{Name: "iam:CreateRole", ResourceARNs: []string{"arn:aws:iam::*:role/*"}},
		{Name: "iam:GetRole", ResourceARNs: []string{"arn:aws:iam::*:role/*"}},
		{Name: "iam:ListRoles", ResourceARNs: []string{"arn:aws:iam::*:role/"}},
		{Name: "iam:DeleteRole", ResourceARNs: []string{"arn:aws:iam::*:role/*"}},
		{Name: "iam:CreatePolicy", ResourceARNs: []string{"arn:aws:iam::*:policy/*"}},
		{Name: "iam:ListPolicies", ResourceARNs: []string{"arn:aws:iam::*:policy/"}},
		{Name: "iam:AttachUserPolicy", ResourceARNs: []string{"arn:aws:iam::*:user/*"}},
		{Name: "iam:AttachRolePolicy", ResourceARNs: []string{"arn:aws:iam::*:role/*"}},
	}
}

func (a *IAMAdapter) HealthCheck() services.Health {
	return services.Health{Status: "healthy"}
}

func (a *IAMAdapter) EmitEvents() []services.EventDef {
	return []services.EventDef{
		{Name: "CreateUser", Source: "iam.amazonaws.com"},
		{Name: "DeleteUser", Source: "iam.amazonaws.com"},
		{Name: "CreateRole", Source: "iam.amazonaws.com"},
		{Name: "AttachUserPolicy", Source: "iam.amazonaws.com"},
	}
}

func (a *IAMAdapter) Metrics() []services.MetricDef {
	return []services.MetricDef{}
}

func (a *IAMAdapter) Quotas() []services.Quota {
	return []services.Quota{
		{Name: "UsersPerAccount", DefaultValue: 5000, Description: "Maximum number of IAM users per account"},
		{Name: "RolesPerAccount", DefaultValue: 1000, Description: "Maximum number of IAM roles per account"},
	}
}

func (a *IAMAdapter) Reset(account, region string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	gatewayIAM.GlobalIAM.Users = make(map[string]*gatewayIAM.User)
	gatewayIAM.GlobalIAM.Roles = make(map[string]*gatewayIAM.Role)
	gatewayIAM.GlobalIAM.AttachedUser = make(map[string][]string)
	gatewayIAM.GlobalIAM.AttachedRole = make(map[string][]string)
	return nil
}

func (a *IAMAdapter) Snapshot() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (a *IAMAdapter) Restore(data map[string]interface{}) error {
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
