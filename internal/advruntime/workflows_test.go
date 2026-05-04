package advruntime

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	workflowservicepb "go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestWorkflowClassifierDetectsPokeEdgeStyleStaleQueue(t *testing.T) {
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	service := &fakeWorkflowService{
		listPages: []*workflowservicepb.ListWorkflowExecutionsResponse{{Executions: manyRunningADVWorkflows(347, "67fe3e95bc2afb49e94cada183986fa1712e47d5", "advance-67fe3e95bc2afb49e94cada183986fa1712e47d5", now.Add(-2*time.Hour))}},
		describe: map[string]*workflowservicepb.DescribeTaskQueueResponse{
			"advance-67fe3e95bc2afb49e94cada183986fa1712e47d5": {
				Pollers: []*taskqueuepb.PollerInfo{},
				Stats:   &taskqueuepb.TaskQueueStats{ApproximateBacklogCount: 12},
			},
		},
	}
	classifier := NewWorkflowClassifier(service, Config{WorkflowStaleAfter: time.Hour}, func() time.Time { return now })

	report, err := classifier.Classify(context.Background())
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}

	if len(report.WorkflowQueues) != 1 {
		t.Fatalf("WorkflowQueues len = %d, want 1", len(report.WorkflowQueues))
	}
	queue := report.WorkflowQueues[0]
	if queue.ProjectID != "67fe3e95bc2afb49e94cada183986fa1712e47d5" {
		t.Fatalf("ProjectID = %q", queue.ProjectID)
	}
	if queue.TaskQueue != "advance-67fe3e95bc2afb49e94cada183986fa1712e47d5" {
		t.Fatalf("TaskQueue = %q", queue.TaskQueue)
	}
	if queue.Status != StatusFail {
		t.Fatalf("Status = %q, want %q", queue.Status, StatusFail)
	}
	if queue.RunningWorkflows != 347 || queue.Pollers != 0 || queue.Backlog != 12 {
		t.Fatalf("queue counts = %#v", queue)
	}
	if queue.OldestRunAge != 2*time.Hour {
		t.Fatalf("OldestRunAge = %s", queue.OldestRunAge)
	}
	if !strings.Contains(queue.Message, "no pollers") || !strings.Contains(queue.Message, "347 running") {
		t.Fatalf("Message = %q", queue.Message)
	}
	if report.Summary.Status != StatusFail || report.Summary.StaleWorkflowQueues != 1 {
		t.Fatalf("Summary = %#v", report.Summary)
	}
}

func TestWorkflowClassifierHandlesHealthyQueueAndPagination(t *testing.T) {
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	service := &fakeWorkflowService{
		listPages: []*workflowservicepb.ListWorkflowExecutionsResponse{
			{Executions: []*workflowpb.WorkflowExecutionInfo{workflowInfo("adv/project/proj1", "advance-proj1", now.Add(-time.Minute))}, NextPageToken: []byte("next")},
			{Executions: []*workflowpb.WorkflowExecutionInfo{workflowInfo("adv/change/proj1/chg1", "advance-proj1", now.Add(-2*time.Minute))}},
		},
		describe: map[string]*workflowservicepb.DescribeTaskQueueResponse{
			"advance-proj1": {Pollers: []*taskqueuepb.PollerInfo{{Identity: "worker-1"}}, Stats: &taskqueuepb.TaskQueueStats{}},
		},
	}
	classifier := NewWorkflowClassifier(service, Config{}, func() time.Time { return now })

	report, err := classifier.Classify(context.Background())
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}

	if len(service.listRequests) != 2 {
		t.Fatalf("list requests = %d, want 2", len(service.listRequests))
	}
	if !strings.Contains(service.listRequests[0].Query, `WorkflowId STARTS WITH "adv/"`) || !strings.Contains(service.listRequests[0].Query, `ExecutionStatus = "Running"`) {
		t.Fatalf("query = %q", service.listRequests[0].Query)
	}
	if string(service.listRequests[1].NextPageToken) != "next" {
		t.Fatalf("next page token = %q", service.listRequests[1].NextPageToken)
	}
	if len(service.describeRequests) != 1 {
		t.Fatalf("describe requests = %d, want 1", len(service.describeRequests))
	}
	describe := service.describeRequests[0]
	if describe.Namespace != DefaultNamespace || describe.GetTaskQueue().GetName() != "advance-proj1" || !describe.ReportStats || describe.TaskQueueType != enumspb.TASK_QUEUE_TYPE_WORKFLOW {
		t.Fatalf("describe request = %#v", describe)
	}
	if got := report.WorkflowQueues[0]; got.Status != StatusPass || got.Pollers != 1 || got.RunningWorkflows != 2 {
		t.Fatalf("queue = %#v", got)
	}
}

func TestWorkflowClassifierFallsBackToProjectQueueFromWorkflowID(t *testing.T) {
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	service := &fakeWorkflowService{
		listPages: []*workflowservicepb.ListWorkflowExecutionsResponse{{Executions: []*workflowpb.WorkflowExecutionInfo{workflowInfo("adv/change/proj2/chg1", "", now)}}},
		describe:  map[string]*workflowservicepb.DescribeTaskQueueResponse{"advance-proj2": {Pollers: []*taskqueuepb.PollerInfo{{Identity: "worker"}}}},
	}
	classifier := NewWorkflowClassifier(service, Config{}, func() time.Time { return now })

	report, err := classifier.Classify(context.Background())
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if got := report.WorkflowQueues[0]; got.ProjectID != "proj2" || got.TaskQueue != "advance-proj2" {
		t.Fatalf("queue = %#v", got)
	}
}

func TestWorkflowClassifierRecordsTemporalErrorsAsFindings(t *testing.T) {
	wantErr := errors.New("temporal unavailable")
	service := &fakeWorkflowService{listErr: wantErr}
	classifier := NewWorkflowClassifier(service, Config{}, time.Now)

	report, err := classifier.Classify(context.Background())
	if err != nil {
		t.Fatalf("Classify should degrade into report, got error: %v", err)
	}
	if report.Summary.Status != StatusWarn || len(report.Findings) != 1 {
		t.Fatalf("report = %#v", report)
	}
	if !strings.Contains(report.Findings[0].Message, "temporal unavailable") {
		t.Fatalf("finding = %#v", report.Findings[0])
	}
}

type fakeWorkflowService struct {
	listPages        []*workflowservicepb.ListWorkflowExecutionsResponse
	listErr          error
	describe         map[string]*workflowservicepb.DescribeTaskQueueResponse
	describeErr      error
	listRequests     []*workflowservicepb.ListWorkflowExecutionsRequest
	describeRequests []*workflowservicepb.DescribeTaskQueueRequest
}

func (f *fakeWorkflowService) ListWorkflowExecutions(ctx context.Context, req *workflowservicepb.ListWorkflowExecutionsRequest) (*workflowservicepb.ListWorkflowExecutionsResponse, error) {
	f.listRequests = append(f.listRequests, req)
	if f.listErr != nil {
		return nil, f.listErr
	}
	idx := len(f.listRequests) - 1
	if idx >= len(f.listPages) {
		return &workflowservicepb.ListWorkflowExecutionsResponse{}, nil
	}
	return f.listPages[idx], nil
}

func (f *fakeWorkflowService) DescribeTaskQueue(ctx context.Context, req *workflowservicepb.DescribeTaskQueueRequest) (*workflowservicepb.DescribeTaskQueueResponse, error) {
	f.describeRequests = append(f.describeRequests, req)
	if f.describeErr != nil {
		return nil, f.describeErr
	}
	if resp, ok := f.describe[req.GetTaskQueue().GetName()]; ok {
		return resp, nil
	}
	return &workflowservicepb.DescribeTaskQueueResponse{}, nil
}

func manyRunningADVWorkflows(count int, projectID, taskQueue string, start time.Time) []*workflowpb.WorkflowExecutionInfo {
	workflows := make([]*workflowpb.WorkflowExecutionInfo, 0, count)
	for i := range count {
		workflows = append(workflows, workflowInfo("adv/change/"+projectID+"/change-"+string(rune('a'+i%26)), taskQueue, start))
	}
	return workflows
}

func workflowInfo(id, taskQueue string, start time.Time) *workflowpb.WorkflowExecutionInfo {
	return &workflowpb.WorkflowExecutionInfo{
		Execution: &commonpb.WorkflowExecution{WorkflowId: id},
		StartTime: timestamppb.New(start),
		Status:    enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
		TaskQueue: taskQueue,
	}
}
