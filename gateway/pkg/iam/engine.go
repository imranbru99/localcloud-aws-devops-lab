package iam

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/imrandev/localcloud/gateway/pkg/context"
)

type Statement struct {
	Sid          string          `json:"Sid,omitempty"`
	Effect       string          `json:"Effect"` // "Allow" or "Deny"
	Action       interface{}     `json:"Action"` // string or []string
	NotAction    interface{}     `json:"NotAction,omitempty"`
	Resource     interface{}     `json:"Resource"` // string or []string
	NotResource  interface{}     `json:"NotResource,omitempty"`
	Condition    map[string]any  `json:"Condition,omitempty"`
}

type PolicyDocument struct {
	Version   string      `json:"Version"`
	Statement []Statement `json:"Statement"`
}

type EvaluationResult struct {
	Allowed        bool     `json:"allowed"`
	MatchedEffect  string   `json:"matched_effect"` // "Allow", "Deny", "ImplicitDeny"
	Reason         string   `json:"reason"`
	MatchedPolicy  string   `json:"matched_policy,omitempty"`
	MatchedStmtSid string   `json:"matched_stmt_sid,omitempty"`
	EvaluationLog  []string `json:"evaluation_log"`
}

// Global in-memory / storage IAM state
type IAMState struct {
	Users         map[string]*User
	Roles         map[string]*Role
	Policies      map[string]*PolicyDocument
	AttachedUser  map[string][]string // username -> []policyArn
	AttachedRole  map[string][]string // rolename -> []policyArn
	Boundaries    map[string]string   // username -> policyArn
	SCPs          []*PolicyDocument
}

type User struct {
	UserName string `json:"user_name"`
	Arn      string `json:"arn"`
	UserId   string `json:"user_id"`
}

type Role struct {
	RoleName string `json:"role_name"`
	Arn      string `json:"arn"`
	RoleId   string `json:"role_id"`
}

var GlobalIAM = &IAMState{
	Users:        make(map[string]*User),
	Roles:        make(map[string]*Role),
	Policies:     make(map[string]*PolicyDocument),
	AttachedUser: make(map[string][]string),
	AttachedRole: make(map[string][]string),
	Boundaries:   make(map[string]string),
	SCPs:         make([]*PolicyDocument, 0),
}

func init() {
	// Register AdministratorAccess policy by default
	adminDoc := &PolicyDocument{
		Version: "2012-10-17",
		Statement: []Statement{
			{
				Sid:      "AdministratorAccess",
				Effect:   "Allow",
				Action:   "*",
				Resource: "*",
			},
		},
	}
	GlobalIAM.Policies["arn:aws:iam::aws:policy/AdministratorAccess"] = adminDoc
}

// Evaluate evaluates an IAM action and resource against principal policies
func Evaluate(ctx *context.RequestContext, action, resourceArn string) EvaluationResult {
	res := EvaluationResult{
		Allowed:       false,
		MatchedEffect: "ImplicitDeny",
		EvaluationLog: []string{},
	}

	res.EvaluationLog = append(res.EvaluationLog, fmt.Sprintf("Evaluating Principal: %s for Action: %s on Resource: %s", ctx.PrincipalARN, action, resourceArn))

	// Root account has full access
	if strings.HasSuffix(ctx.PrincipalARN, ":root") {
		res.Allowed = true
		res.MatchedEffect = "Allow"
		res.Reason = "Root credentials have implicit full administrative access"
		res.EvaluationLog = append(res.EvaluationLog, res.Reason)
		return res
	}

	// 1. Check SCPs (if any)
	for idx, scp := range GlobalIAM.SCPs {
		allowedBySCP := false
		for _, stmt := range scp.Statement {
			if matchesAction(stmt.Action, action) && matchesResource(stmt.Resource, resourceArn) {
				if strings.EqualFold(stmt.Effect, "Deny") {
					res.Allowed = false
					res.MatchedEffect = "Deny"
					res.Reason = fmt.Sprintf("Explicit deny by Service Control Policy (SCP) index %d, Sid: %s", idx, stmt.Sid)
					res.EvaluationLog = append(res.EvaluationLog, res.Reason)
					return res
				}
				if strings.EqualFold(stmt.Effect, "Allow") {
					allowedBySCP = true
				}
			}
		}
		if !allowedBySCP {
			res.Allowed = false
			res.MatchedEffect = "ImplicitDeny"
			res.Reason = "Request not explicitly allowed by Organization Service Control Policies"
			return res
		}
	}

	// 2. Identity policies (attached to user or role)
	var policiesToCheck []*PolicyDocument
	// Check policies attached to caller
	var userOrRoleName string
	if strings.Contains(ctx.PrincipalARN, ":user/") {
		parts := strings.Split(ctx.PrincipalARN, ":user/")
		userOrRoleName = parts[len(parts)-1]
		for _, polArn := range GlobalIAM.AttachedUser[userOrRoleName] {
			if doc, ok := GlobalIAM.Policies[polArn]; ok {
				policiesToCheck = append(policiesToCheck, doc)
			}
		}
	} else if strings.Contains(ctx.PrincipalARN, ":role/") {
		parts := strings.Split(ctx.PrincipalARN, ":role/")
		userOrRoleName = parts[len(parts)-1]
		for _, polArn := range GlobalIAM.AttachedRole[userOrRoleName] {
			if doc, ok := GlobalIAM.Policies[polArn]; ok {
				policiesToCheck = append(policiesToCheck, doc)
			}
		}
	}

	// If no custom policies attached, default to Admin for local dev
	if len(policiesToCheck) == 0 {
		policiesToCheck = append(policiesToCheck, GlobalIAM.Policies["arn:aws:iam::aws:policy/AdministratorAccess"])
	}

	hasExplicitAllow := false
	for _, doc := range policiesToCheck {
		for _, stmt := range doc.Statement {
			if matchesAction(stmt.Action, action) && matchesResource(stmt.Resource, resourceArn) {
				if strings.EqualFold(stmt.Effect, "Deny") {
					res.Allowed = false
					res.MatchedEffect = "Deny"
					res.MatchedStmtSid = stmt.Sid
					res.Reason = fmt.Sprintf("Explicit deny in statement Sid: %s", stmt.Sid)
					res.EvaluationLog = append(res.EvaluationLog, res.Reason)
					return res
				}
				if strings.EqualFold(stmt.Effect, "Allow") {
					hasExplicitAllow = true
					res.MatchedStmtSid = stmt.Sid
					res.EvaluationLog = append(res.EvaluationLog, fmt.Sprintf("Allowed by statement Sid: %s", stmt.Sid))
				}
			}
		}
	}

	if hasExplicitAllow {
		res.Allowed = true
		res.MatchedEffect = "Allow"
		res.Reason = "Explicitly allowed by identity policy"
		return res
	}

	res.Allowed = false
	res.MatchedEffect = "ImplicitDeny"
	res.Reason = fmt.Sprintf("No identity-based policy allows the %s action on resource %s", action, resourceArn)
	res.EvaluationLog = append(res.EvaluationLog, res.Reason)
	return res
}

func matchesAction(actionPattern interface{}, targetAction string) bool {
	patterns := toStringSlice(actionPattern)
	for _, p := range patterns {
		if p == "*" || strings.EqualFold(p, targetAction) {
			return true
		}
		// Wildcard matching e.g. s3:* or s3:Get*
		if strings.HasSuffix(p, "*") {
			prefix := strings.TrimSuffix(p, "*")
			if strings.HasPrefix(strings.ToLower(targetAction), strings.ToLower(prefix)) {
				return true
			}
		}
	}
	return false
}

func matchesResource(resourcePattern interface{}, targetResource string) bool {
	patterns := toStringSlice(resourcePattern)
	for _, p := range patterns {
		if p == "*" || p == targetResource {
			return true
		}
		// Wildcard match
		matched, _ := path.Match(p, targetResource)
		if matched {
			return true
		}
		if strings.HasSuffix(p, "*") {
			prefix := strings.TrimSuffix(p, "*")
			if strings.HasPrefix(targetResource, prefix) {
				return true
			}
		}
	}
	return false
}

func toStringSlice(val interface{}) []string {
	if val == nil {
		return []string{}
	}
	switch v := val.(type) {
	case string:
		return []string{v}
	case []interface{}:
		var out []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return v
	default:
		return []string{}
	}
}

// ParsePolicyDocument parses raw JSON into PolicyDocument
func ParsePolicyDocument(raw string) (*PolicyDocument, error) {
	var doc PolicyDocument
	err := json.Unmarshal([]byte(raw), &doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}
