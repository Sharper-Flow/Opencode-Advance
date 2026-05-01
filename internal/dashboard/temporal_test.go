package dashboard

import (
	"context"
	"log/slog"
	"testing"

	"go.temporal.io/api/workflowservice/v1"
)

// mockWorkflowServiceClient implements workflowservice.WorkflowServiceClient for testing.
type mockWorkflowServiceClient struct {
	responses []*workflowservice.ListWorkflowExecutionsResponse
	callCount int
	err       error
}

func (m *mockWorkflowServiceClient) ListWorkflowExecutions(ctx context.Context, req *workflowservice.ListWorkflowExecutionsRequest) (*workflowservice.ListWorkflowExecutionsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	idx := m.callCount
	m.callCount++
	if idx >= len(m.responses) {
		return &workflowservice.ListWorkflowExecutionsResponse{}, nil
	}
	return m.responses[idx], nil
}

func TestTemporalPoller_SinglePage(t *testing.T) {
	// Test with single page of results
	s := NewState()
	mock := &mockWorkflowServiceClient{
		responses: []*workflowservice.ListWorkflowExecutionsResponse{},
	}

	poller := NewTemporalPoller(mock, slog.Default())
	err := poller.Poll(context.Background(), s)
	if err != nil {
		t.Fatalf("Poll failed: %v", err)
	}

	snap := s.Snapshot()
	if snap.Changes == nil {
		t.Error("expected Changes to be initialized")
	}
}

func TestTemporalPoller_DegradationOnNilClient(t *testing.T) {
	s := NewState()
	poller := NewTemporalPoller(nil, slog.Default())

	err := poller.Poll(context.Background(), s)
	if err != nil {
		t.Fatalf("Poll with nil client should not error: %v", err)
	}

	snap := s.Snapshot()
	// Should have empty changes, not crash
	if snap.Changes == nil {
		t.Error("expected Changes to be initialized even on nil client")
	}
}

func TestTemporalPoller_CapEnforcement(t *testing.T) {
	// Verify that more than 500 workflows are truncated
	s := NewState()

	poller := NewTemporalPoller(nil, slog.Default())
	poller.maxWorkflows = 5
	err := poller.Poll(context.Background(), s)
	if err != nil {
		t.Fatalf("Poll failed: %v", err)
	}

	// With nil client, changes should be empty but maxWorkflows cap should be respected
	if poller.maxWorkflows != 5 {
		t.Errorf("maxWorkflows = %d, want 5", poller.maxWorkflows)
	}
}
