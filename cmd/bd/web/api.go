package web

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/steveyegge/beads/internal/types"
)

// API response structures
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type UpdateIssueRequest struct {
	Title              *string `json:"title,omitempty"`
	Description        *string `json:"description,omitempty"`
	Design             *string `json:"design,omitempty"`
	AcceptanceCriteria *string `json:"acceptance_criteria,omitempty"`
	Notes              *string `json:"notes,omitempty"`
	Status             *string `json:"status,omitempty"`
	Priority           *int    `json:"priority,omitempty"`
	IssueType          *string `json:"issue_type,omitempty"`
	Assignee           *string `json:"assignee,omitempty"`
}

type AddDependencyRequest struct {
	DependsOnID string `json:"depends_on_id"`
	Type        string `json:"type"`
}

type RemoveDependencyRequest struct {
	DependsOnID string `json:"depends_on_id"`
	Type        string `json:"type"`
}

// APIListIssues returns issues as JSON
func (h *Handlers) APIListIssues(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	filters := parseListFilters(r)

	issueFilter := types.IssueFilter{
		Limit: 1000, // Higher limit for API
	}

	if filters.Status != "" && filters.Status != "all" {
		status := types.Status(filters.Status)
		issueFilter.Status = &status
	}

	issues, err := h.server.store.SearchIssues(ctx, "", issueFilter)
	if err != nil {
		h.jsonError(w, "Failed to load issues", http.StatusInternalServerError)
		return
	}

	h.jsonSuccess(w, issues)
}

// APIGetIssue returns a single issue as JSON
func (h *Handlers) APIGetIssue(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	issueID := chi.URLParam(r, "id")

	issue, err := h.server.store.GetIssue(ctx, issueID)
	if err != nil {
		h.jsonError(w, "Failed to load issue", http.StatusInternalServerError)
		return
	}
	if issue == nil {
		h.jsonError(w, "Issue not found", http.StatusNotFound)
		return
	}

	// Include dependencies and labels
	labels, _ := h.server.store.GetLabels(ctx, issueID)
	deps, _ := h.server.store.GetDependencyRecords(ctx, issueID)

	response := struct {
		*types.Issue
		Labels       []string            `json:"labels,omitempty"`
		Dependencies []*types.Dependency `json:"dependencies,omitempty"`
	}{
		Issue:        issue,
		Labels:       labels,
		Dependencies: deps,
	}

	h.jsonSuccess(w, response)
}

// APICreateIssue creates a new issue via JSON API
func (h *Handlers) APICreateIssue(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var issue types.Issue
	if err := json.NewDecoder(r.Body).Decode(&issue); err != nil {
		h.jsonError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Set defaults
	if issue.Status == "" {
		issue.Status = types.StatusOpen
	}
	if issue.Priority < 0 || issue.Priority > 4 {
		issue.Priority = 2
	}

	if err := h.server.store.CreateIssue(ctx, &issue, h.server.actor); err != nil {
		h.jsonError(w, "Failed to create issue", http.StatusInternalServerError)
		return
	}

	h.jsonSuccess(w, issue)
}

// APIUpdateIssue updates an issue via JSON API
func (h *Handlers) APIUpdateIssue(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	issueID := chi.URLParam(r, "id")

	var req UpdateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Build updates map
	updates := make(map[string]interface{})

	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Design != nil {
		updates["design"] = *req.Design
	}
	if req.AcceptanceCriteria != nil {
		updates["acceptance_criteria"] = *req.AcceptanceCriteria
	}
	if req.Notes != nil {
		updates["notes"] = *req.Notes
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.IssueType != nil {
		updates["issue_type"] = *req.IssueType
	}
	if req.Assignee != nil {
		updates["assignee"] = *req.Assignee
	}

	if len(updates) == 0 {
		h.jsonError(w, "No updates provided", http.StatusBadRequest)
		return
	}

	if err := h.server.store.UpdateIssue(ctx, issueID, updates, h.server.actor); err != nil {
		h.jsonError(w, "Failed to update issue", http.StatusInternalServerError)
		return
	}

	// Return updated issue
	issue, _ := h.server.store.GetIssue(ctx, issueID)
	h.jsonSuccess(w, issue)
}

// APIAddDependency adds a dependency to an issue
func (h *Handlers) APIAddDependency(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	issueID := chi.URLParam(r, "id")

	var req AddDependencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate dependency type
	depType := types.DependencyType(req.Type)
	if !depType.IsValid() {
		h.jsonError(w, "Invalid dependency type", http.StatusBadRequest)
		return
	}

	// Create dependency
	dep := &types.Dependency{
		IssueID:     issueID,
		DependsOnID: req.DependsOnID,
		Type:        depType,
	}

	if err := h.server.store.AddDependency(ctx, dep, h.server.actor); err != nil {
		h.jsonError(w, "Failed to add dependency", http.StatusInternalServerError)
		return
	}

	h.jsonSuccess(w, map[string]string{"message": "Dependency added"})
}

// APIRemoveDependency removes a dependency from an issue
func (h *Handlers) APIRemoveDependency(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	issueID := chi.URLParam(r, "id")

	var req RemoveDependencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.server.store.RemoveDependency(ctx, issueID, req.DependsOnID, h.server.actor); err != nil {
		h.jsonError(w, "Failed to remove dependency", http.StatusInternalServerError)
		return
	}

	h.jsonSuccess(w, map[string]string{"message": "Dependency removed"})
}

// APISearchIssues provides autocomplete search
func (h *Handlers) APISearchIssues(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	query := r.URL.Query().Get("q")

	// Search by ID or title
	issues, err := h.server.store.SearchIssues(ctx, query, types.IssueFilter{
		Limit: 20, // Limit for autocomplete
	})
	if err != nil {
		h.jsonError(w, "Search failed", http.StatusInternalServerError)
		return
	}

	// Return simplified results
	results := make([]map[string]string, 0)
	for _, issue := range issues {
		results = append(results, map[string]string{
			"id":    issue.ID,
			"title": issue.Title,
		})
	}

	h.jsonSuccess(w, results)
}

// Helper functions

func (h *Handlers) jsonSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
	})
}

func (h *Handlers) jsonError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	})
}

// parseQueryInt parses an integer from query parameters
func parseQueryInt(r *http.Request, key string, defaultVal int) int {
	if val := r.URL.Query().Get(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}
