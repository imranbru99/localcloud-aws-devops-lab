package console

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/imrandev/localcloud/gateway/pkg/cloudtrail"
	"github.com/imrandev/localcloud/gateway/pkg/context"
	gatewayIAM "github.com/imrandev/localcloud/gateway/pkg/iam"
	"github.com/imrandev/localcloud/services"
)

type ConsoleHandler struct {
	mu sync.RWMutex
}

func NewHandler() *ConsoleHandler {
	return &ConsoleHandler{}
}

func (h *ConsoleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// API endpoints for console UI
	if strings.HasPrefix(path, "/api/") {
		h.handleAPI(w, r)
		return
	}

	// Serve Console Single Page Web Application
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(IndexHTML))
}

func (h *ConsoleHandler) handleAPI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch path {
	case "/api/stats":
		h.handleStats(w, r)
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
	case "/api/reset":
		h.handleReset(w, r)
	default:
		http.NotFound(w, r)
	}
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

	// Send initial connect ping
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
		"currency": "USD",
		"total_estimated_monthly": "$0.00 (LocalCloud Free Learning)",
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
