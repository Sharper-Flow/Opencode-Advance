package health

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/advruntime"
	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
	operatorservicepb "go.temporal.io/api/operatorservice/v1"
	workflowservicepb "go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
)

func init() {
	registerBuiltin("adv-runtime", CheckAdvRuntime)
}

// CheckAdvRuntime runs ADV runtime diagnostics and maps findings to health.Check
// rows. It degrades gracefully when Temporal or the OpenCode DB is unavailable.
func CheckAdvRuntime(ctx context.Context, stack *cfg.Stack, opts Options) ([]Check, error) {
	checks := []Check{}
	config := advruntime.Config{
		Address:            temporalAddress(stack),
		Namespace:          temporalNamespace(stack),
		SessionStaleAfter:  5 * time.Minute,
		WorktreeStaleAfter: 7 * 24 * time.Hour,
	}

	// --- Temporal checks (workflow visibility + search attributes) ---
	if stack.Temporal != nil && stack.Temporal.Address != "" {
		provider := advruntime.NewTemporalClientProvider(config, nil)
		defer provider.Close()

		client, err := provider.Client(ctx)
		if err != nil {
			checks = append(checks, Check{
				Name:    "adv-runtime.temporal",
				Status:  StatusWarn,
				Message: fmt.Sprintf("Temporal unreachable: %v", err),
				Hint:    "start Temporal dev server with 'oca temporal start' or verify stack.toml [temporal] address",
			})
		} else {
			// Search attributes
			checker := advruntime.NewSearchAttributeChecker(&operatorServiceAdapter{client.OperatorService()}, config)
			report, _ := checker.Check(ctx)
			for _, attr := range report.SearchAttributes {
				checks = append(checks, mapSearchAttribute(attr))
			}

			// Workflow / task queue classifier
			classifier := advruntime.NewWorkflowClassifier(&workflowServiceAdapter{client.WorkflowService()}, config, nil)
			report, _ = classifier.Classify(ctx)
			for _, queue := range report.WorkflowQueues {
				checks = append(checks, mapWorkflowQueue(queue))
			}
			for _, finding := range report.Findings {
				checks = append(checks, mapFinding(finding))
			}
		}
	} else {
		checks = append(checks, Check{
			Name:    "adv-runtime.temporal",
			Status:  StatusPass,
			Message: "Temporal not configured in stack.toml — ADV runtime checks skipped",
		})
	}

	// --- Session debt scan ---
	dbPath := defaultOpenCodeDBPath()
	if _, err := os.Stat(dbPath); err == nil {
		db, err := sql.Open("sqlite", dbPath)
		if err == nil {
			defer db.Close()
			scanner := advruntime.NewSessionDebtScanner(db, config)
			finding, _ := scanner.Scan(ctx)
			checks = append(checks, mapSessionDebt(finding, dbPath))
		}
	} else {
		checks = append(checks, Check{
			Name:    "adv-runtime.session-debt",
			Status:  StatusPass,
			Message: "OpenCode session DB not found — skipping session debt scan",
		})
	}

	// --- Worktree census ---
	worktreeRoot := defaultWorktreeRoot()
	advRoot := defaultADVStateRoot()
	census := advruntime.NewWorktreeCensus(worktreeRoot, advRoot, config)
	worktrees, _ := census.Scan(ctx)
	for _, wt := range worktrees {
		checks = append(checks, mapWorktree(wt))
	}

	// --- Recovery plan synthesis (informational) ---
	if len(checks) > 0 {
		report := buildReportFromChecks(checks, config)
		synth := advruntime.NewRecoveryPlanSynthesizer()
		plan := synth.Synthesize(report)
		if len(plan) > 0 {
			checks = append(checks, Check{
				Name:    "adv-runtime.recovery-plan",
				Status:  StatusWarn,
				Message: fmt.Sprintf("%d recovery step(s) available — run 'oca adv recover --dry-run'", len(plan)),
				Hint:    "review the recovery plan before applying any changes",
			})
		}
	}

	return checks, nil
}

func mapSearchAttribute(attr advruntime.SearchAttributeStatus) Check {
	status := StatusPass
	switch attr.Status {
	case advruntime.StatusFail:
		status = StatusFail
	case advruntime.StatusWarn, advruntime.StatusUnknown:
		status = StatusWarn
	}
	return Check{
		Name:    "adv-runtime.search-attribute." + attr.Name,
		Status:  status,
		Message: attr.Message,
		Hint:    attr.Hint,
	}
}

func mapWorkflowQueue(queue advruntime.WorkflowQueueStatus) Check {
	status := StatusPass
	switch queue.Status {
	case advruntime.StatusFail:
		status = StatusFail
	case advruntime.StatusWarn:
		status = StatusWarn
	}
	msg := queue.Message
	if msg == "" {
		msg = fmt.Sprintf("queue %s: %d poller(s), %d running workflow(s)", queue.TaskQueue, queue.Pollers, queue.RunningWorkflows)
	}
	return Check{
		Name:    "adv-runtime.queue." + queue.TaskQueue,
		Status:  status,
		Message: msg,
		Hint:    queue.Hint,
	}
}

func mapFinding(finding advruntime.Finding) Check {
	status := StatusPass
	switch finding.Severity {
	case advruntime.SeverityError:
		status = StatusFail
	case advruntime.SeverityWarn:
		status = StatusWarn
	}
	return Check{
		Name:    "adv-runtime.finding." + finding.Code,
		Status:  status,
		Message: finding.Message,
		Hint:    finding.Hint,
	}
}

func mapSessionDebt(debt advruntime.SessionDebtFinding, dbPath string) Check {
	status := StatusPass
	if debt.Status == advruntime.StatusWarn {
		status = StatusWarn
	}
	msg := debt.Message
	if msg == "" {
		msg = fmt.Sprintf("session DB %s: %d repairable row(s), %d ignored", dbPath, debt.Repairable, debt.Ignored)
	}
	return Check{
		Name:    "adv-runtime.session-debt",
		Status:  status,
		Message: msg,
		Hint:    debt.Hint,
	}
}

func mapWorktree(wt advruntime.WorktreeFinding) Check {
	status := StatusPass
	if wt.Status == advruntime.StatusWarn {
		status = StatusWarn
	}
	name := "adv-runtime.worktree." + filepath.Base(wt.Path)
	if wt.Branch != "" {
		name = "adv-runtime.worktree." + wt.Branch
	}
	return Check{
		Name:    name,
		Status:  status,
		Message: wt.Message,
		Hint:    wt.Hint,
	}
}

func buildReportFromChecks(checks []Check, config advruntime.Config) advruntime.Report {
	report := advruntime.NewReport(config)
	for _, c := range checks {
		switch c.Status {
		case StatusFail:
			report.Summary.Status = advruntime.StatusFail
		case StatusWarn:
			if report.Summary.Status == advruntime.StatusPass {
				report.Summary.Status = advruntime.StatusWarn
			}
		}
	}
	return report
}

func temporalAddress(stack *cfg.Stack) string {
	if stack.Temporal != nil && stack.Temporal.Address != "" {
		return stack.Temporal.Address
	}
	return advruntime.DefaultAddress
}

func temporalNamespace(stack *cfg.Stack) string {
	if stack.Temporal != nil && stack.Temporal.Namespace != "" {
		return stack.Temporal.Namespace
	}
	return advruntime.DefaultNamespace
}

func defaultOpenCodeDBPath() string {
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "opencode", "opencode.db")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "opencode", "opencode.db")
}

func defaultWorktreeRoot() string {
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "opencode", "worktree")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "opencode", "worktree")
}

func defaultADVStateRoot() string {
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "opencode", "plugins", "advance")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "opencode", "plugins", "advance")
}

// operatorServiceAdapter adapts the gRPC client interface (which has variadic
// grpc.CallOption) to the narrow advruntime.OperatorService interface.
type operatorServiceAdapter struct {
	client interface {
		ListSearchAttributes(context.Context, *operatorservicepb.ListSearchAttributesRequest, ...grpc.CallOption) (*operatorservicepb.ListSearchAttributesResponse, error)
	}
}

func (a *operatorServiceAdapter) ListSearchAttributes(ctx context.Context, req *operatorservicepb.ListSearchAttributesRequest) (*operatorservicepb.ListSearchAttributesResponse, error) {
	return a.client.ListSearchAttributes(ctx, req)
}

// workflowServiceAdapter adapts the gRPC client interface to the narrow
// advruntime.WorkflowService interface.
type workflowServiceAdapter struct {
	client interface {
		ListWorkflowExecutions(context.Context, *workflowservicepb.ListWorkflowExecutionsRequest, ...grpc.CallOption) (*workflowservicepb.ListWorkflowExecutionsResponse, error)
		DescribeTaskQueue(context.Context, *workflowservicepb.DescribeTaskQueueRequest, ...grpc.CallOption) (*workflowservicepb.DescribeTaskQueueResponse, error)
	}
}

func (a *workflowServiceAdapter) ListWorkflowExecutions(ctx context.Context, req *workflowservicepb.ListWorkflowExecutionsRequest) (*workflowservicepb.ListWorkflowExecutionsResponse, error) {
	return a.client.ListWorkflowExecutions(ctx, req)
}

func (a *workflowServiceAdapter) DescribeTaskQueue(ctx context.Context, req *workflowservicepb.DescribeTaskQueueRequest) (*workflowservicepb.DescribeTaskQueueResponse, error) {
	return a.client.DescribeTaskQueue(ctx, req)
}
