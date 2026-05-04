package advruntime

import (
	"context"
	"errors"
	"testing"
	"time"

	operatorservice "go.temporal.io/api/operatorservice/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

func TestDefaultThresholdsAreStableAndNonZero(t *testing.T) {
	thresholds := DefaultThresholds()

	if thresholds.MaxWorkflows != 500 {
		t.Fatalf("MaxWorkflows = %d, want 500", thresholds.MaxWorkflows)
	}
	if thresholds.WorkflowStaleAfter <= 0 {
		t.Fatalf("WorkflowStaleAfter must be positive")
	}
	if thresholds.SessionStaleAfter <= 0 {
		t.Fatalf("SessionStaleAfter must be positive")
	}
	if thresholds.WorktreeStaleAfter <= 0 {
		t.Fatalf("WorktreeStaleAfter must be positive")
	}
}

func TestNewReportInitializesCollectionsAndSummary(t *testing.T) {
	report := NewReport(Config{Address: "127.0.0.1:7233", Namespace: "default"})

	if report.GeneratedAt.IsZero() {
		t.Fatalf("GeneratedAt was not set")
	}
	if report.Config.Address != "127.0.0.1:7233" || report.Config.Namespace != "default" {
		t.Fatalf("Config = %#v", report.Config)
	}
	if report.Summary.Status != StatusPass {
		t.Fatalf("Summary.Status = %q, want %q", report.Summary.Status, StatusPass)
	}
	if report.WorkflowQueues == nil || report.SearchAttributes == nil || report.SessionDebt == nil || report.Worktrees == nil || report.Findings == nil {
		t.Fatalf("report collections must be non-nil: %#v", report)
	}
}

func TestTemporalClientProviderDialsLazilyAndReusesClient(t *testing.T) {
	ctx := context.Background()
	fake := &fakeTemporalClient{}
	var gotOptions client.Options
	dials := 0
	provider := NewTemporalClientProvider(Config{Address: "127.0.0.1:7233", Namespace: "adv-test"}, func(ctx context.Context, opts client.Options) (TemporalClient, error) {
		dials++
		gotOptions = opts
		return fake, nil
	})

	if dials != 0 {
		t.Fatalf("provider dialed eagerly")
	}

	first, err := provider.Client(ctx)
	if err != nil {
		t.Fatalf("Client first call: %v", err)
	}
	second, err := provider.Client(ctx)
	if err != nil {
		t.Fatalf("Client second call: %v", err)
	}
	if first != second {
		t.Fatalf("provider did not reuse client")
	}
	if dials != 1 {
		t.Fatalf("dials = %d, want 1", dials)
	}
	if gotOptions.HostPort != "127.0.0.1:7233" || gotOptions.Namespace != "adv-test" {
		t.Fatalf("dial options = %#v", gotOptions)
	}

	provider.Close()
	provider.Close()
	if fake.closeCount != 1 {
		t.Fatalf("Close count = %d, want 1", fake.closeCount)
	}
}

func TestTemporalClientProviderUsesDefaultsAndReturnsDialErrors(t *testing.T) {
	wantErr := errors.New("dial failed")
	var gotOptions client.Options
	provider := NewTemporalClientProvider(Config{}, func(ctx context.Context, opts client.Options) (TemporalClient, error) {
		gotOptions = opts
		return nil, wantErr
	})

	got, err := provider.Client(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if got != nil {
		t.Fatalf("client = %#v, want nil", got)
	}
	if gotOptions.HostPort != DefaultAddress || gotOptions.Namespace != DefaultNamespace {
		t.Fatalf("dial options = %#v", gotOptions)
	}
}

func TestFakeTemporalClientProvider(t *testing.T) {
	wantClient := &fakeTemporalClient{}
	provider := NewFakeTemporalClientProvider(wantClient, nil)

	got, err := provider.Client(context.Background())
	if err != nil {
		t.Fatalf("Client: %v", err)
	}
	if got != wantClient {
		t.Fatalf("client = %#v, want %#v", got, wantClient)
	}
	provider.Close()
	if wantClient.closeCount != 0 {
		t.Fatalf("fake provider must not close injected client")
	}

	wantErr := errors.New("unavailable")
	errProvider := NewFakeTemporalClientProvider(nil, wantErr)
	if _, err := errProvider.Client(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("error provider err = %v, want %v", err, wantErr)
	}
}

func TestConfigWithDefaults(t *testing.T) {
	config := Config{WorkflowStaleAfter: time.Hour}.WithDefaults()

	if config.Address != DefaultAddress || config.Namespace != DefaultNamespace {
		t.Fatalf("config defaults = %#v", config)
	}
	if config.WorkflowStaleAfter != time.Hour {
		t.Fatalf("explicit WorkflowStaleAfter overwritten")
	}
	if config.SessionStaleAfter == 0 || config.WorktreeStaleAfter == 0 || config.MaxWorkflows == 0 {
		t.Fatalf("threshold defaults missing: %#v", config)
	}
}

type fakeTemporalClient struct {
	closeCount int
}

func (f *fakeTemporalClient) WorkflowService() workflowservice.WorkflowServiceClient { return nil }

func (f *fakeTemporalClient) OperatorService() operatorservice.OperatorServiceClient { return nil }

func (f *fakeTemporalClient) Close() { f.closeCount++ }
