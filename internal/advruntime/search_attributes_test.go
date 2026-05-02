package advruntime

import (
	"context"
	"errors"
	"strings"
	"testing"

	enumspb "go.temporal.io/api/enums/v1"
	operatorservicepb "go.temporal.io/api/operatorservice/v1"
)

func TestSearchAttributeCheckerPassesWhenRequiredAttributesExist(t *testing.T) {
	service := &fakeOperatorService{response: &operatorservicepb.ListSearchAttributesResponse{CustomAttributes: map[string]enumspb.IndexedValueType{
		"AdvProjectId":      enumspb.INDEXED_VALUE_TYPE_KEYWORD,
		"AdvChangeId":       enumspb.INDEXED_VALUE_TYPE_KEYWORD,
		"AdvChangeStatus":   enumspb.INDEXED_VALUE_TYPE_KEYWORD,
		"AdvActiveGate":     enumspb.INDEXED_VALUE_TYPE_KEYWORD,
		"AdvDoomLoopActive": enumspb.INDEXED_VALUE_TYPE_BOOL,
	}}}
	checker := NewSearchAttributeChecker(service, Config{Namespace: "adv-test"})

	report, err := checker.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	if len(service.requests) != 1 || service.requests[0].Namespace != "adv-test" {
		t.Fatalf("requests = %#v", service.requests)
	}
	if report.Summary.Status != StatusPass {
		t.Fatalf("summary = %#v", report.Summary)
	}
	for _, attr := range report.SearchAttributes {
		if attr.Status != StatusPass {
			t.Fatalf("attr = %#v", attr)
		}
	}
}

func TestSearchAttributeCheckerDetectsMissingAttributes(t *testing.T) {
	service := &fakeOperatorService{response: &operatorservicepb.ListSearchAttributesResponse{CustomAttributes: map[string]enumspb.IndexedValueType{
		"AdvProjectId": enumspb.INDEXED_VALUE_TYPE_KEYWORD,
	}}}
	checker := NewSearchAttributeChecker(service, Config{})

	report, err := checker.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	if report.Summary.Status != StatusFail || report.Summary.MissingSearchAttributes != 4 {
		t.Fatalf("summary = %#v", report.Summary)
	}
	missing := searchAttributeByName(report, "AdvChangeId")
	if missing.Status != StatusFail || !strings.Contains(missing.Message, "missing") {
		t.Fatalf("missing attr = %#v", missing)
	}
}

func TestSearchAttributeCheckerDetectsWrongTypes(t *testing.T) {
	service := &fakeOperatorService{response: &operatorservicepb.ListSearchAttributesResponse{CustomAttributes: map[string]enumspb.IndexedValueType{
		"AdvProjectId":      enumspb.INDEXED_VALUE_TYPE_TEXT,
		"AdvChangeId":       enumspb.INDEXED_VALUE_TYPE_KEYWORD,
		"AdvChangeStatus":   enumspb.INDEXED_VALUE_TYPE_KEYWORD,
		"AdvActiveGate":     enumspb.INDEXED_VALUE_TYPE_KEYWORD,
		"AdvDoomLoopActive": enumspb.INDEXED_VALUE_TYPE_KEYWORD,
	}}}
	checker := NewSearchAttributeChecker(service, Config{})

	report, err := checker.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	if report.Summary.Status != StatusFail {
		t.Fatalf("summary = %#v", report.Summary)
	}
	wrong := searchAttributeByName(report, "AdvProjectId")
	if wrong.Status != StatusFail || wrong.Actual != "INDEXED_VALUE_TYPE_TEXT" || wrong.Expected != "INDEXED_VALUE_TYPE_KEYWORD" {
		t.Fatalf("wrong attr = %#v", wrong)
	}
}

func TestSearchAttributeCheckerDegradesWhenAPIUnsupported(t *testing.T) {
	service := &fakeOperatorService{err: errors.New("ListSearchAttributes is not supported by this visibility store")}
	checker := NewSearchAttributeChecker(service, Config{})

	report, err := checker.Check(context.Background())
	if err != nil {
		t.Fatalf("Check should degrade into report, got error: %v", err)
	}

	if report.Summary.Status != StatusWarn {
		t.Fatalf("summary = %#v", report.Summary)
	}
	if len(report.SearchAttributes) != len(RequiredSearchAttributes()) {
		t.Fatalf("attrs len = %d", len(report.SearchAttributes))
	}
	for _, attr := range report.SearchAttributes {
		if attr.Status != StatusUnknown || !strings.Contains(attr.Message, "visibility unknown") {
			t.Fatalf("attr = %#v", attr)
		}
	}
}

type fakeOperatorService struct {
	response *operatorservicepb.ListSearchAttributesResponse
	err      error
	requests []*operatorservicepb.ListSearchAttributesRequest
}

func (f *fakeOperatorService) ListSearchAttributes(ctx context.Context, req *operatorservicepb.ListSearchAttributesRequest) (*operatorservicepb.ListSearchAttributesResponse, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}
	if f.response == nil {
		return &operatorservicepb.ListSearchAttributesResponse{}, nil
	}
	return f.response, nil
}

func searchAttributeByName(report Report, name string) SearchAttributeStatus {
	for _, attr := range report.SearchAttributes {
		if attr.Name == name {
			return attr
		}
	}
	return SearchAttributeStatus{}
}
