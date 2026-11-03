package console

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/imrandev/localcloud/gateway/pkg/cloudtrail"
	"github.com/imrandev/localcloud/gateway/pkg/context"
	gatewayIAM "github.com/imrandev/localcloud/gateway/pkg/iam"
	"github.com/imrandev/localcloud/services"
	"github.com/imrandev/localcloud/services/cloudformation"
	"github.com/imrandev/localcloud/services/ec2"
	"github.com/imrandev/localcloud/services/rds"
	"github.com/imrandev/localcloud/services/s3"
	"github.com/imrandev/localcloud/services/ses"
	"github.com/imrandev/localcloud/services/sns"
	"github.com/imrandev/localcloud/services/sqs"
)

type ConsoleHandler struct {
	mu          sync.RWMutex
	chaosConfig map[string]interface{}
}

func NewHandler() *ConsoleHandler {
	return &ConsoleHandler{
		chaosConfig: map[string]interface{}{
			"latency_ms":         0,
			"simulate_az_outage": false,
			"throttle_requests":  false,
		},
	}
}

func (h *ConsoleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if path == "/static/tailwind.min.js" {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(TailwindJS))
		return
	}

	if strings.HasPrefix(path, "/api/") {
		h.handleAPI(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(IndexHTML))
}

func (h *ConsoleHandler) handleAPI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch path {
	case "/api/stats":
		h.handleStats(w, r)
	case "/api/resources":
		h.handleResources(w, r)
	case "/api/stream/inspector":
		h.handleStreamInspector(w, r)
	case "/api/events":
		h.handleEvents(w, r)
	case "/api/iam/simulate":
		h.handleIAMSimulate(w, r)
	case "/api/labs":
		h.handleLabs(w, r)
	case "/api/labs/check":
		h.handleLabCheck(w, r)
	case "/api/billing":
		h.handleBilling(w, r)
	case "/api/mailbox":
		h.handleMailbox(w, r)
	case "/api/chaos":
		h.handleChaos(w, r)
	case "/api/quizzes":
		h.handleQuizzes(w, r)
	case "/api/resourcemap":
		h.handleResourceMap(w, r)
	case "/api/reset":
		h.handleReset(w, r)
	case "/api/cloudshell":
		h.handleCloudShell(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *ConsoleHandler) handleResources(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	res := map[string]interface{}{
		"ec2":            ec2.GetInstances(),
		"sqs":            sqs.GetQueues(),
		"sns":            sns.GetTopics(),
		"rds":            rds.GetDBInstances(),
		"cloudformation": cloudformation.GetStacks(),
		"s3":             s3.GetBuckets(),
		"iam":            gatewayIAM.GlobalIAM.Users,
	}
	_ = json.NewEncoder(w).Encode(res)
}

func (h *ConsoleHandler) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stats := map[string]interface{}{
		"active_region":  "us-east-1",
		"account_id":     "000000000000",
		"account_name":   "LocalCloud Root",
		"iam_users":      len(gatewayIAM.GlobalIAM.Users),
		"iam_roles":      len(gatewayIAM.GlobalIAM.Roles),
		"services_count": len(services.All()),
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	}
	_ = json.NewEncoder(w).Encode(stats)
}

func (h *ConsoleHandler) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	events := cloudtrail.Pipeline.GetRecentEvents(100)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}

func (h *ConsoleHandler) handleStreamInspector(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := cloudtrail.Pipeline.Subscribe()
	defer cloudtrail.Pipeline.Unsubscribe(ch)

	fmt.Fprintf(w, "data: %s\n\n", `{"type":"connected","msg":"Live API Inspector Stream Connected"}`)
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case ev := <-ch:
			b, _ := json.Marshal(ev)
			fmt.Fprintf(w, "data: %s\n\n", string(b))
			flusher.Flush()
		}
	}
}

func (h *ConsoleHandler) handleIAMSimulate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		PolicyDocument string `json:"policy_document"`
		Action         string `json:"action"`
		ResourceArn    string `json:"resource_arn"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.Action == "" {
		req.Action = "s3:GetObject"
	}
	if req.ResourceArn == "" {
		req.ResourceArn = "arn:aws:s3:::mybucket/file.txt"
	}

	doc, err := gatewayIAM.ParsePolicyDocument(req.PolicyDocument)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid policy JSON: " + err.Error(),
		})
		return
	}

	testArn := "arn:aws:iam::000000000000:policy/SimulatedPolicy"
	gatewayIAM.GlobalIAM.Policies[testArn] = doc
	gatewayIAM.GlobalIAM.AttachedUser["simulator"] = []string{testArn}

	reqCtx := &context.RequestContext{
		AccountID:    "000000000000",
		PrincipalARN: "arn:aws:iam::000000000000:user/simulator",
	}
	res := gatewayIAM.Evaluate(reqCtx, req.Action, req.ResourceArn)

	_ = json.NewEncoder(w).Encode(res)
}

func (h *ConsoleHandler) handleBilling(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	billing := map[string]interface{}{
		"currency":                    "USD",
		"total_estimated_monthly":     "$0.00 (LocalCloud Free Learning)",
		"real_aws_equivalent_monthly": "$34.50",
		"services": []map[string]string{
			{"service": "Amazon EC2", "units": "1 t3.micro (730 hrs)", "real_aws_cost": "$7.45", "localcloud_cost": "$0.00"},
			{"service": "Amazon S3", "units": "Standard Storage + Requests", "real_aws_cost": "$2.30", "localcloud_cost": "$0.00"},
			{"service": "Amazon RDS", "units": "db.t3.micro PostgreSQL", "real_aws_cost": "$18.25", "localcloud_cost": "$0.00"},
			{"service": "AWS Lambda", "units": "1,000,000 Invocations", "real_aws_cost": "$0.20", "localcloud_cost": "$0.00"},
			{"service": "Amazon CloudWatch", "units": "Metrics & Alarms", "real_aws_cost": "$6.30", "localcloud_cost": "$0.00"},
		},
		"well_architected_tip": "In real AWS, use AWS Budgets, CloudWatch Billing Alarms, and Cost Allocation Tags to avoid surprise bills.",
	}
	_ = json.NewEncoder(w).Encode(billing)
}

func (h *ConsoleHandler) handleMailbox(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ses.GetMailbox())
}

func (h *ConsoleHandler) handleChaos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost {
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		h.mu.Lock()
		for k, v := range req {
			h.chaosConfig[k] = v
		}
		h.mu.Unlock()
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	_ = json.NewEncoder(w).Encode(h.chaosConfig)
}

func (h *ConsoleHandler) handleQuizzes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	quizzes := []map[string]interface{}{
		{
			"id":          "q1",
			"track":       "Cloud Practitioner (CLF-C02)",
			"question":    "Which AWS pillar of the Well-Architected Framework focuses on running workloads effectively and gaining insights into operations?",
			"options":     []string{"Operational Excellence", "Security", "Reliability", "Cost Optimization"},
			"correct":     0,
			"explanation": "Operational Excellence focuses on running and monitoring systems to deliver business value and continually improving processes.",
		},
		{
			"id":          "q2",
			"track":       "Solutions Architect Associate (SAA-C03)",
			"question":    "An application requires decoupled asynchronous processing where multiple subscriber services receive every order event. Which architecture is best?",
			"options":     []string{"Amazon S3 with Lifecycle Rules", "Amazon SNS Topic with Amazon SQS Queue Fanout", "Single EC2 instance with cron", "Amazon Route 53 Geolocation"},
			"correct":     1,
			"explanation": "The SNS topic to multiple SQS queues (fanout pattern) delivers identical broadcast messages to decoupled subscriber queues.",
		},
		{
			"id":       "q3",
			"track":    "Security Specialty",
			"question": "What is the primary difference between a Security Group and a Network Access Control List (NACL)?",
			"options": []string{
				"Security Groups are stateless; NACLs are stateful",
				"Security Groups operate at instance/ENI level and are stateful; NACLs operate at subnet level and are stateless",
				"NACLs only support Allow rules",
				"Security Groups support Deny rules explicitly",
			},
			"correct":     1,
			"explanation": "Security Groups are stateful firewalls evaluated at the instance ENI level. NACLs are stateless subnets firewalls evaluating rules in numeric order.",
		},
	}
	_ = json.NewEncoder(w).Encode(quizzes)
}

func (h *ConsoleHandler) handleResourceMap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resMap := map[string]interface{}{
		"nodes": []map[string]string{
			{"id": "vpc-1", "label": "VPC (10.0.0.0/16)", "type": "vpc"},
			{"id": "sub-pub", "label": "Public Subnet (10.0.1.0/24)", "type": "subnet"},
			{"id": "sub-priv", "label": "Private Subnet (10.0.2.0/24)", "type": "subnet"},
			{"id": "ec2-app", "label": "EC2 App Server (t3.micro)", "type": "instance"},
			{"id": "rds-db", "label": "RDS PostgreSQL Database", "type": "database"},
			{"id": "s3-store", "label": "S3 Data Storage Bucket", "type": "storage"},
		},
		"edges": []map[string]string{
			{"from": "vpc-1", "to": "sub-pub"},
			{"from": "vpc-1", "to": "sub-priv"},
			{"from": "sub-pub", "to": "ec2-app"},
			{"from": "sub-priv", "to": "rds-db"},
			{"from": "ec2-app", "to": "s3-store"},
		},
	}
	_ = json.NewEncoder(w).Encode(resMap)
}

func (h *ConsoleHandler) handleReset(w http.ResponseWriter, r *http.Request) {
	_ = services.ResetAll("000000000000", "us-east-1")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "LocalCloud state reset successfully"})
}

func (h *ConsoleHandler) handleLabs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(GetAllLabs())
}

func (h *ConsoleHandler) handleLabCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req struct {
		LabID string `json:"lab_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	result := CheckLab(req.LabID)
	_ = json.NewEncoder(w).Encode(result)
}

func (h *ConsoleHandler) handleCloudShell(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Command string `json:"command"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	cmdStr := strings.TrimSpace(req.Command)
	if cmdStr == "" {
		_ = json.NewEncoder(w).Encode(map[string]string{"output": ""})
		return
	}

	// Auto-prefix endpoint URL for AWS CLI commands if not specified
	if strings.HasPrefix(cmdStr, "aws ") && !strings.Contains(cmdStr, "--endpoint-url") {
		cmdStr = strings.Replace(cmdStr, "aws ", "aws --endpoint-url http://localhost:4566 ", 1)
	}

	ctx, cancel := r.Context(), func() {}
	_ = ctx
	_ = cancel

	cmd := exec.Command("bash", "-c", cmdStr)
	cmd.Dir = "/home/imran/project/aws"
	cmd.Env = append(os.Environ(),
		"AWS_ENDPOINT_URL=http://localhost:4566",
		"AWS_ACCESS_KEY_ID=test",
		"AWS_SECRET_ACCESS_KEY=test",
		"AWS_DEFAULT_REGION=us-east-1",
		"PATH=/home/imran/.local/bin:/usr/local/bin:/usr/bin:/bin:"+os.Getenv("PATH"),
	)

	out, err := cmd.CombinedOutput()
	output := string(out)
	if err != nil && output == "" {
		output = fmt.Sprintf("Error: %v", err)
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"output": output,
	})
}
