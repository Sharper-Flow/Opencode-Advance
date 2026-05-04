package advruntime

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	workflowservicepb "go.temporal.io/api/workflowservice/v1"
)

const advRunningWorkflowQuery = `WorkflowId STARTS WITH "adv/" AND ExecutionStatus = "Running"`

// WorkflowService is the narrow Temporal workflow service surface used by the
// classifier. workflowservice.WorkflowServiceClient satisfies this interface.
type WorkflowService interface {
	ListWorkflowExecutions(context.Context, *workflowservicepb.ListWorkflowExecutionsRequest) (*workflowservicepb.ListWorkflowExecutionsResponse, error)
	DescribeTaskQueue(context.Context, *workflowservicepb.DescribeTaskQueueRequest) (*workflowservicepb.DescribeTaskQueueResponse, error)
}

type WorkflowClassifier struct {
	service WorkflowService
	config  Config
	now     func() time.Time
}

func NewWorkflowClassifier(service WorkflowService, config Config, now func() time.Time) *WorkflowClassifier {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &WorkflowClassifier{service: service, config: config.WithDefaults(), now: now}
}

func (c *WorkflowClassifier) Classify(ctx context.Context) (Report, error) {
	report := NewReport(c.config)
	if c.service == nil {
		addFinding(&report, SeverityWarn, "ADV_TEMPORAL_UNAVAILABLE", "Temporal workflow service unavailable", "start Temporal or run recovery in dry-run mode")
		return report, nil
	}

	workflows, err := c.listADVWorkflows(ctx)
	if err != nil {
		addFinding(&report, SeverityWarn, "ADV_WORKFLOW_LIST_FAILED", fmt.Sprintf("list ADV workflows failed: %v", err), "check Temporal reachability and namespace")
		return report, nil
	}

	groups := groupWorkflowsByQueue(workflows)
	queues := make([]string, 0, len(groups))
	for queue := range groups {
		queues = append(queues, queue)
	}
	sort.Strings(queues)

	for _, queue := range queues {
		report.WorkflowQueues = append(report.WorkflowQueues, c.describeQueue(ctx, queue, groups[queue]))
	}

	report.Summary.WorkflowQueues = len(report.WorkflowQueues)
	for _, queue := range report.WorkflowQueues {
		if queue.Status == StatusFail {
			report.Summary.StaleWorkflowQueues++
		}
	}
	refreshSummaryStatus(&report)
	return report, nil
}

func (c *WorkflowClassifier) listADVWorkflows(ctx context.Context) ([]*workflowpb.WorkflowExecutionInfo, error) {
	var workflows []*workflowpb.WorkflowExecutionInfo
	var nextPageToken []byte
	for len(workflows) < c.config.MaxWorkflows {
		resp, err := c.service.ListWorkflowExecutions(ctx, &workflowservicepb.ListWorkflowExecutionsRequest{
			Namespace:     c.config.Namespace,
			PageSize:      int32(min(c.config.MaxWorkflows-len(workflows), 100)),
			NextPageToken: nextPageToken,
			Query:         advRunningWorkflowQuery,
		})
		if err != nil {
			return nil, err
		}
		for _, wf := range resp.GetExecutions() {
			if len(workflows) >= c.config.MaxWorkflows {
				break
			}
			if isADVRunningWorkflow(wf) {
				workflows = append(workflows, wf)
			}
		}
		if len(resp.GetNextPageToken()) == 0 {
			break
		}
		nextPageToken = resp.GetNextPageToken()
	}
	return workflows, nil
}

func (c *WorkflowClassifier) describeQueue(ctx context.Context, queue string, workflows []*workflowpb.WorkflowExecutionInfo) WorkflowQueueStatus {
	status := WorkflowQueueStatus{
		ProjectID:        projectIDForQueue(queue, workflows),
		TaskQueue:        queue,
		Status:           StatusPass,
		RunningWorkflows: len(workflows),
		OldestRunAge:     oldestRunAge(workflows, c.now()),
	}

	resp, err := c.service.DescribeTaskQueue(ctx, &workflowservicepb.DescribeTaskQueueRequest{
		Namespace:     c.config.Namespace,
		TaskQueue:     &taskqueuepb.TaskQueue{Name: queue},
		TaskQueueType: enumspb.TASK_QUEUE_TYPE_WORKFLOW,
		ReportStats:   true,
	})
	if err != nil {
		status.Status = StatusWarn
		status.Message = fmt.Sprintf("describe task queue failed: %v", err)
		status.Hint = "check Temporal task queue visibility"
		return status
	}

	status.Pollers = len(resp.GetPollers())
	if stats := resp.GetStats(); stats != nil {
		status.Backlog = stats.GetApproximateBacklogCount()
	}

	if status.RunningWorkflows > 0 && status.Pollers == 0 {
		status.Status = StatusWarn
		status.Message = fmt.Sprintf("task queue has no pollers and %d running workflows", status.RunningWorkflows)
		status.Hint = "restart OpenCode/Advance worker for this project queue"
		if status.OldestRunAge >= c.config.WorkflowStaleAfter {
			status.Status = StatusFail
			status.Message = fmt.Sprintf("task queue has no pollers and %d running workflows; oldest run age %s exceeds %s", status.RunningWorkflows, status.OldestRunAge.Round(time.Second), c.config.WorkflowStaleAfter)
		}
	}

	return status
}

func groupWorkflowsByQueue(workflows []*workflowpb.WorkflowExecutionInfo) map[string][]*workflowpb.WorkflowExecutionInfo {
	groups := map[string][]*workflowpb.WorkflowExecutionInfo{}
	for _, wf := range workflows {
		projectID := projectIDFromWorkflowID(workflowID(wf))
		queue := wf.GetTaskQueue()
		if queue == "" && projectID != "" {
			queue = "advance-" + projectID
		}
		if queue == "" {
			continue
		}
		groups[queue] = append(groups[queue], wf)
	}
	return groups
}

func isADVRunningWorkflow(wf *workflowpb.WorkflowExecutionInfo) bool {
	return strings.HasPrefix(workflowID(wf), "adv/") && wf.GetStatus() == enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING
}

func workflowID(wf *workflowpb.WorkflowExecutionInfo) string {
	if wf == nil || wf.GetExecution() == nil {
		return ""
	}
	return wf.GetExecution().GetWorkflowId()
}

func projectIDForQueue(queue string, workflows []*workflowpb.WorkflowExecutionInfo) string {
	for _, wf := range workflows {
		if projectID := projectIDFromWorkflowID(workflowID(wf)); projectID != "" {
			return projectID
		}
	}
	return strings.TrimPrefix(queue, "advance-")
}

func projectIDFromWorkflowID(id string) string {
	parts := strings.Split(id, "/")
	if len(parts) >= 3 && parts[0] == "adv" && parts[1] == "project" {
		return parts[2]
	}
	if len(parts) >= 4 && parts[0] == "adv" && parts[1] == "change" {
		return parts[2]
	}
	return ""
}

func oldestRunAge(workflows []*workflowpb.WorkflowExecutionInfo, now time.Time) time.Duration {
	var oldest time.Time
	for _, wf := range workflows {
		start := wf.GetStartTime().AsTime()
		if start.IsZero() {
			continue
		}
		if oldest.IsZero() || start.Before(oldest) {
			oldest = start
		}
	}
	if oldest.IsZero() || now.Before(oldest) {
		return 0
	}
	return now.Sub(oldest)
}

func addFinding(report *Report, severity Severity, code, message, hint string) {
	report.Findings = append(report.Findings, Finding{Code: code, Severity: severity, Message: message, Hint: hint})
	report.Summary.Findings = len(report.Findings)
	refreshSummaryStatus(report)
}

func refreshSummaryStatus(report *Report) {
	status := StatusPass
	for _, finding := range report.Findings {
		if finding.Severity == SeverityError {
			status = StatusFail
		} else if status == StatusPass && finding.Severity == SeverityWarn {
			status = StatusWarn
		}
	}
	for _, queue := range report.WorkflowQueues {
		if queue.Status == StatusFail {
			status = StatusFail
		} else if status == StatusPass && queue.Status == StatusWarn {
			status = StatusWarn
		}
	}
	for _, attr := range report.SearchAttributes {
		if attr.Status == StatusFail {
			status = StatusFail
		} else if status == StatusPass && (attr.Status == StatusWarn || attr.Status == StatusUnknown) {
			status = StatusWarn
		}
	}
	report.Summary.Status = status
}
