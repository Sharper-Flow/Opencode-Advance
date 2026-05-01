package dashboard

import (
	"sort"
	"strings"
)

// filterParams holds query-parameter filter criteria.
type filterParams struct {
	Project string
	Status  string
	Query   string
}

// filterChanges applies filter params to a change slice.
func filterChanges(changes []ChangeRow, p filterParams) []ChangeRow {
	var result []ChangeRow
	for _, c := range changes {
		if p.Project != "" && c.Project != p.Project {
			continue
		}
		if p.Status != "" && c.Status != p.Status {
			continue
		}
		if p.Query != "" {
			q := strings.ToLower(p.Query)
			if !strings.Contains(strings.ToLower(c.Title), q) &&
				!strings.Contains(strings.ToLower(c.ID), q) {
				continue
			}
		}
		result = append(result, c)
	}
	return result
}

// sortChanges sorts changes by the given field ("title", "project", "status").
// Invalid field returns the original slice unchanged.
func sortChanges(changes []ChangeRow, field string) []ChangeRow {
	if len(changes) <= 1 {
		return changes
	}

	sorted := make([]ChangeRow, len(changes))
	copy(sorted, changes)

	switch field {
	case "title":
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Title < sorted[j].Title
		})
	case "project":
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Project < sorted[j].Project
		})
	case "status":
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Status < sorted[j].Status
		})
	default:
		return changes
	}
	return sorted
}
