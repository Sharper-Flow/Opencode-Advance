package dashboard

import (
	"context"
	"log/slog"
	"os"

	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
)

const (
	defaultMaxWorkflows = 500
	defaultPageSize     = 100
	defaultNamespace    = "default"
	changeQueryPrefix   = `WorkflowId STARTS WITH "adv/change/"`
)

// workflowLister is the narrow interface for listing workflows, extracted for testability.
type workflowLister interface {
	ListWorkflowExecutions(ctx context.Context, req *workflowservice.ListWorkflowExecutionsRequest) (*workflowservice.ListWorkflowExecutionsResponse, error)
}

// TemporalPoller implements Poller by fetching ADV change workflows from Temporal.
type TemporalPoller struct {
	client       workflowLister
	namespace    string
	maxWorkflows int
	logger       *slog.Logger
}

// NewTemporalPoller creates a new Temporal poller.
func NewTemporalPoller(client workflowLister, logger *slog.Logger) *TemporalPoller {
	if logger == nil {
		logger = slog.Default()
	}
	ns := os.Getenv("TEMPORAL_NAMESPACE")
	if ns == "" {
		ns = defaultNamespace
	}
	return &TemporalPoller{
		client:       client,
		namespace:    ns,
		maxWorkflows: defaultMaxWorkflows,
		logger:       logger,
	}
}

// Poll fetches all ADV change workflows from Temporal and updates the state.
// Returns nil even on Temporal errors (graceful degradation).
func (p *TemporalPoller) Poll(ctx context.Context, s *State) error {
	if p.client == nil {
		// No Temporal client — set empty changes and return (degradation)
		s.Update(func(snap *Snapshot) {
			snap.Changes = []ChangeRow{}
		})
		return nil
	}

	var allChanges []ChangeRow
	allChanges = make([]ChangeRow, 0) // ensure non-nil
	var nextPageToken []byte

	for len(allChanges) < p.maxWorkflows {
		resp, err := p.client.ListWorkflowExecutions(ctx, &workflowservice.ListWorkflowExecutionsRequest{
			Namespace:     p.namespace,
			PageSize:      defaultPageSize,
			NextPageToken: nextPageToken,
			Query:         changeQueryPrefix,
		})
		if err != nil {
			p.logger.Warn("temporal list workflows failed", "error", err)
			s.Update(func(snap *Snapshot) {
				snap.Changes = []ChangeRow{}
			})
			return nil // graceful degradation
		}

		for _, wf := range resp.Executions {
			if len(allChanges) >= p.maxWorkflows {
				break
			}
			row := p.mapWorkflowToRow(wf)
			allChanges = append(allChanges, row)
		}

		if len(resp.NextPageToken) == 0 {
			break
		}
		nextPageToken = resp.NextPageToken
	}

	if len(allChanges) >= p.maxWorkflows {
		p.logger.Info("workflow list capped", "cap", p.maxWorkflows)
	}

	s.Update(func(snap *Snapshot) {
		snap.Changes = allChanges
	})
	return nil
}

// mapWorkflowToRow converts a Temporal workflow execution to a ChangeRow.
func (p *TemporalPoller) mapWorkflowToRow(wf *workflow.WorkflowExecutionInfo) ChangeRow {
	row := ChangeRow{
		Status: "unknown",
	}

	if wf.Execution != nil {
		row.ID = wf.Execution.WorkflowId
	}

	// Extract search attributes
	if wf.SearchAttributes != nil {
		for _, sa := range wf.SearchAttributes.IndexedFields {
			_ = sa // TODO: parse search attributes in follow-up
		}
	}

	// Map status
	row.Status = wf.Status.String()

	return row
}
