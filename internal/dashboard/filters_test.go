package dashboard

import (
	"testing"
)

func TestFilter_ByProject(t *testing.T) {
	changes := []ChangeRow{
		{ID: "ch-1", Title: "Alpha", Status: "active", Project: "proj-a"},
		{ID: "ch-2", Title: "Beta", Status: "active", Project: "proj-b"},
		{ID: "ch-3", Title: "Gamma", Status: "archived", Project: "proj-a"},
	}
	result := filterChanges(changes, filterParams{Project: "proj-a"})
	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
	for _, c := range result {
		if c.Project != "proj-a" {
			t.Errorf("got project %q", c.Project)
		}
	}
}

func TestFilter_ByStatus(t *testing.T) {
	changes := []ChangeRow{
		{ID: "ch-1", Title: "Alpha", Status: "active"},
		{ID: "ch-2", Title: "Beta", Status: "archived"},
	}
	result := filterChanges(changes, filterParams{Status: "active"})
	if len(result) != 1 || result[0].ID != "ch-1" {
		t.Errorf("expected ch-1, got %+v", result)
	}
}

func TestFilter_ByQuery(t *testing.T) {
	changes := []ChangeRow{
		{ID: "ch-alpha-123", Title: "Fix auth bug"},
		{ID: "ch-beta-456", Title: "Add dashboard feature"},
	}
	result := filterChanges(changes, filterParams{Query: "auth"})
	if len(result) != 1 || result[0].Title != "Fix auth bug" {
		t.Errorf("expected auth change, got %+v", result)
	}
}

func TestFilter_QueryMatchesID(t *testing.T) {
	changes := []ChangeRow{
		{ID: "ch-alpha-123", Title: "Something"},
		{ID: "ch-beta-456", Title: "Other"},
	}
	result := filterChanges(changes, filterParams{Query: "alpha"})
	if len(result) != 1 || result[0].ID != "ch-alpha-123" {
		t.Errorf("expected ch-alpha-123, got %+v", result)
	}
}

func TestFilter_Combined(t *testing.T) {
	changes := []ChangeRow{
		{ID: "ch-1", Title: "Auth fix", Status: "active", Project: "proj-a"},
		{ID: "ch-2", Title: "Auth refactor", Status: "draft", Project: "proj-a"},
		{ID: "ch-3", Title: "Auth add", Status: "active", Project: "proj-b"},
	}
	result := filterChanges(changes, filterParams{Project: "proj-a", Status: "active", Query: "auth"})
	if len(result) != 1 || result[0].ID != "ch-1" {
		t.Errorf("expected ch-1, got %+v", result)
	}
}

func TestFilter_EmptyResult(t *testing.T) {
	changes := []ChangeRow{
		{ID: "ch-1", Title: "Alpha", Status: "active"},
	}
	result := filterChanges(changes, filterParams{Status: "nonexistent"})
	if len(result) != 0 {
		t.Errorf("expected 0, got %d", len(result))
	}
}

func TestFilter_NoParams(t *testing.T) {
	changes := []ChangeRow{
		{ID: "ch-1", Title: "Alpha"},
		{ID: "ch-2", Title: "Beta"},
	}
	result := filterChanges(changes, filterParams{})
	if len(result) != 2 {
		t.Errorf("expected all 2, got %d", len(result))
	}
}

func TestSort_ByTitle(t *testing.T) {
	changes := []ChangeRow{
		{ID: "ch-1", Title: "Zebra"},
		{ID: "ch-2", Title: "Alpha"},
		{ID: "ch-3", Title: "Middle"},
	}
	result := sortChanges(changes, "title")
	if result[0].Title != "Alpha" || result[2].Title != "Zebra" {
		var names []string
		for _, r := range result {
			names = append(names, r.Title)
		}
		t.Errorf("expected Alpha, Middle, Zebra; got %v", names)
	}
}

func TestSort_ByProject(t *testing.T) {
	changes := []ChangeRow{
		{ID: "ch-1", Title: "A", Project: "z-proj"},
		{ID: "ch-2", Title: "B", Project: "a-proj"},
	}
	result := sortChanges(changes, "project")
	if result[0].Project != "a-proj" {
		t.Errorf("expected a-proj first, got %s", result[0].Project)
	}
}

func TestSort_InvalidField(t *testing.T) {
	changes := []ChangeRow{
		{ID: "ch-1", Title: "A"},
	}
	result := sortChanges(changes, "invalid")
	if len(result) != 1 {
		t.Error("invalid sort should return original slice")
	}
}
