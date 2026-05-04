package advruntime

import (
	"context"
	"fmt"
	"sort"
	"strings"

	enumspb "go.temporal.io/api/enums/v1"
	operatorservicepb "go.temporal.io/api/operatorservice/v1"
)

// OperatorService is the narrow Temporal operator service surface used by the
// search-attribute checker. operatorservice.OperatorServiceClient satisfies it.
type OperatorService interface {
	ListSearchAttributes(context.Context, *operatorservicepb.ListSearchAttributesRequest) (*operatorservicepb.ListSearchAttributesResponse, error)
}

type SearchAttributeChecker struct {
	service OperatorService
	config  Config
}

func NewSearchAttributeChecker(service OperatorService, config Config) *SearchAttributeChecker {
	return &SearchAttributeChecker{service: service, config: config.WithDefaults()}
}

func RequiredSearchAttributes() map[string]enumspb.IndexedValueType {
	return map[string]enumspb.IndexedValueType{
		"AdvProjectId":      enumspb.INDEXED_VALUE_TYPE_KEYWORD,
		"AdvChangeId":       enumspb.INDEXED_VALUE_TYPE_KEYWORD,
		"AdvChangeStatus":   enumspb.INDEXED_VALUE_TYPE_KEYWORD,
		"AdvActiveGate":     enumspb.INDEXED_VALUE_TYPE_KEYWORD,
		"AdvDoomLoopActive": enumspb.INDEXED_VALUE_TYPE_BOOL,
	}
}

func (c *SearchAttributeChecker) Check(ctx context.Context) (Report, error) {
	report := NewReport(c.config)
	if c.service == nil {
		c.markVisibilityUnknown(&report, "Temporal operator service unavailable")
		return report, nil
	}

	resp, err := c.service.ListSearchAttributes(ctx, &operatorservicepb.ListSearchAttributesRequest{Namespace: c.config.Namespace})
	if err != nil {
		message := fmt.Sprintf("search attribute visibility unknown: %v", err)
		if isUnsupportedSearchAttributeError(err) {
			message = fmt.Sprintf("search attribute visibility unknown: ListSearchAttributes unsupported by this Temporal visibility store: %v", err)
		}
		c.markVisibilityUnknown(&report, message)
		return report, nil
	}

	required := RequiredSearchAttributes()
	for _, name := range sortedRequiredSearchAttributeNames(required) {
		expected := required[name]
		actual, ok := resp.GetCustomAttributes()[name]
		attr := SearchAttributeStatus{
			Name:     name,
			Expected: indexedValueTypeName(expected),
			Status:   StatusPass,
		}
		if !ok {
			attr.Status = StatusFail
			attr.Message = fmt.Sprintf("required search attribute %s missing", name)
			attr.Hint = "register Advance Temporal search attributes before relying on visibility queries"
			report.Summary.MissingSearchAttributes++
		} else {
			attr.Actual = indexedValueTypeName(actual)
			if actual != expected {
				attr.Status = StatusFail
				attr.Message = fmt.Sprintf("required search attribute %s has type %s, want %s", name, indexedValueTypeName(actual), indexedValueTypeName(expected))
				attr.Hint = "recreate local Temporal visibility schema or repair Advance search attributes"
			}
		}
		report.SearchAttributes = append(report.SearchAttributes, attr)
	}

	refreshSummaryStatus(&report)
	return report, nil
}

func (c *SearchAttributeChecker) markVisibilityUnknown(report *Report, message string) {
	for _, name := range sortedRequiredSearchAttributeNames(RequiredSearchAttributes()) {
		report.SearchAttributes = append(report.SearchAttributes, SearchAttributeStatus{
			Name:     name,
			Expected: indexedValueTypeName(RequiredSearchAttributes()[name]),
			Status:   StatusUnknown,
			Message:  "search attribute visibility unknown",
			Hint:     "Temporal may be using local visibility or an unsupported API; verify with Advance diagnostics",
		})
	}
	addFinding(report, SeverityWarn, "ADV_SEARCH_ATTRIBUTES_UNKNOWN", message, "continue with degraded warning and avoid treating visibility queries as authoritative")
}

func indexedValueTypeName(value enumspb.IndexedValueType) string {
	switch value {
	case enumspb.INDEXED_VALUE_TYPE_TEXT:
		return "INDEXED_VALUE_TYPE_TEXT"
	case enumspb.INDEXED_VALUE_TYPE_KEYWORD:
		return "INDEXED_VALUE_TYPE_KEYWORD"
	case enumspb.INDEXED_VALUE_TYPE_INT:
		return "INDEXED_VALUE_TYPE_INT"
	case enumspb.INDEXED_VALUE_TYPE_DOUBLE:
		return "INDEXED_VALUE_TYPE_DOUBLE"
	case enumspb.INDEXED_VALUE_TYPE_BOOL:
		return "INDEXED_VALUE_TYPE_BOOL"
	case enumspb.INDEXED_VALUE_TYPE_DATETIME:
		return "INDEXED_VALUE_TYPE_DATETIME"
	case enumspb.INDEXED_VALUE_TYPE_KEYWORD_LIST:
		return "INDEXED_VALUE_TYPE_KEYWORD_LIST"
	default:
		return "INDEXED_VALUE_TYPE_UNSPECIFIED"
	}
}

func sortedRequiredSearchAttributeNames(attrs map[string]enumspb.IndexedValueType) []string {
	names := make([]string, 0, len(attrs))
	for name := range attrs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func isUnsupportedSearchAttributeError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not supported") || strings.Contains(msg, "unsupported")
}
