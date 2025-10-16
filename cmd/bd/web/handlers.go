package web

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/steveyegge/beads/internal/types"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	server *Server
}

// NewHandlers creates a new handlers instance
func NewHandlers(s *Server) *Handlers {
	return &Handlers{server: s}
}

// PageData represents common data passed to templates
type PageData struct {
	Title       string
	CurrentPath string
	Error       string
	Success     string
	Data        interface{}
}

// ListPageData represents data for the list page
type ListPageData struct {
	Issues      []*types.Issue
	Filters     ListFilters
	Total       int
	CurrentPage int
	TotalPages  int
}

// DetailPageData represents data for the detail page
type DetailPageData struct {
	Issue             *types.Issue
	Labels            []string
	Dependencies      []*DependencyInfo
	Dependents        []*DependencyInfo
	DependencyTypes   []types.DependencyType
	IssueTypes        []types.IssueType
	Statuses          []types.Status
	Priorities        []int
}

// DependencyInfo contains enriched dependency information
type DependencyInfo struct {
	Issue *types.Issue
	Type  types.DependencyType
}

// ListFilters represents filter options for the list view
type ListFilters struct {
	Status         string
	Priority       string
	Type           string
	Assignee       string
	ParentID       string
	ChildID        string
	DiscoveredFrom string
	Sort           string
	Order          string
	Page           int
}

// RedirectToIssues redirects to the issues list page
func (h *Handlers) RedirectToIssues(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/issues", http.StatusSeeOther)
}

// ListIssues displays the issues list page
func (h *Handlers) ListIssues(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	filters := parseListFilters(r)

	// Build issue filter from query params
	issueFilter := types.IssueFilter{
		Limit: 50, // Default page size
	}

	if filters.Status != "" && filters.Status != "all" {
		status := types.Status(filters.Status)
		issueFilter.Status = &status
	}

	if filters.Priority != "" && filters.Priority != "all" {
		priority, err := strconv.Atoi(filters.Priority)
		if err == nil {
			issueFilter.Priority = &priority
		}
	}

	if filters.Type != "" && filters.Type != "all" {
		issueType := types.IssueType(filters.Type)
		issueFilter.IssueType = &issueType
	}

	if filters.Assignee != "" {
		issueFilter.Assignee = &filters.Assignee
	}

	// Get issues
	issues, err := h.server.store.SearchIssues(ctx, "", issueFilter)
	if err != nil {
		h.renderError(w, r, "Failed to load issues", err)
		return
	}

	// Apply client-side filtering for dependencies (until we extend storage)
	issues = h.filterByDependencies(ctx, issues, filters)

	// Apply sorting
	issues = sortIssues(issues, filters.Sort, filters.Order)

	// Prepare page data
	pageData := PageData{
		Title:       "Issues",
		CurrentPath: "/issues",
		Data: ListPageData{
			Issues:      issues,
			Filters:     filters,
			Total:       len(issues),
			CurrentPage: filters.Page,
			TotalPages:  (len(issues) + 49) / 50, // Ceiling division
		},
	}

	h.render(w, "list.html", pageData)
}

// IssueDetail displays a single issue's detail page
func (h *Handlers) IssueDetail(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	issueID := chi.URLParam(r, "issueID")

	// Validate issue ID format
	if !isValidIssueID(issueID) {
		http.NotFound(w, r)
		return
	}

	// Get issue
	issue, err := h.server.store.GetIssue(ctx, issueID)
	if err != nil {
		h.renderError(w, r, "Failed to load issue", err)
		return
	}
	if issue == nil {
		http.NotFound(w, r)
		return
	}

	// Get labels
	labels, _ := h.server.store.GetLabels(ctx, issueID)

	// Get dependencies with type information
	depRecords, _ := h.server.store.GetDependencyRecords(ctx, issueID)
	dependencies := make([]*DependencyInfo, 0)
	for _, dep := range depRecords {
		depIssue, err := h.server.store.GetIssue(ctx, dep.DependsOnID)
		if err == nil && depIssue != nil {
			dependencies = append(dependencies, &DependencyInfo{
				Issue: depIssue,
				Type:  dep.Type,
			})
		}
	}

	// Get dependents (reverse dependencies)
	dependents := make([]*DependencyInfo, 0)
	allDepRecords, _ := h.server.store.GetAllDependencyRecords(ctx)
	for _, deps := range allDepRecords {
		for _, dep := range deps {
			if dep.DependsOnID == issueID {
				depIssue, err := h.server.store.GetIssue(ctx, dep.IssueID)
				if err == nil && depIssue != nil {
					dependents = append(dependents, &DependencyInfo{
						Issue: depIssue,
						Type:  dep.Type,
					})
				}
			}
		}
	}

	// Prepare page data
	pageData := PageData{
		Title:       issue.Title,
		CurrentPath: "/" + issueID,
		Data: DetailPageData{
			Issue:        issue,
			Labels:       labels,
			Dependencies: dependencies,
			Dependents:   dependents,
			DependencyTypes: []types.DependencyType{
				types.DepBlocks,
				types.DepRelated,
				types.DepParentChild,
				types.DepDiscoveredFrom,
			},
			IssueTypes: []types.IssueType{
				types.TypeBug,
				types.TypeFeature,
				types.TypeTask,
				types.TypeEpic,
				types.TypeChore,
			},
			Statuses: []types.Status{
				types.StatusOpen,
				types.StatusInProgress,
				types.StatusBlocked,
				types.StatusClosed,
			},
			Priorities: []int{0, 1, 2, 3, 4},
		},
	}

	h.render(w, "detail.html", pageData)
}

// NewIssue displays the create issue form
func (h *Handlers) NewIssue(w http.ResponseWriter, r *http.Request) {
	pageData := PageData{
		Title:       "Create Issue",
		CurrentPath: "/issues/new",
		Data: struct {
			IssueTypes []types.IssueType
			Priorities []int
		}{
			IssueTypes: []types.IssueType{
				types.TypeBug,
				types.TypeFeature,
				types.TypeTask,
				types.TypeEpic,
				types.TypeChore,
			},
			Priorities: []int{0, 1, 2, 3, 4},
		},
	}

	h.render(w, "create.html", pageData)
}

// CreateIssue handles form submission to create a new issue
func (h *Handlers) CreateIssue(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Parse form
	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, "Invalid form data", err)
		return
	}

	// Build issue from form
	priority, _ := strconv.Atoi(r.FormValue("priority"))
	if priority < 0 || priority > 4 {
		priority = 2
	}

	issue := &types.Issue{
		Title:              r.FormValue("title"),
		Description:        r.FormValue("description"),
		Design:             r.FormValue("design"),
		AcceptanceCriteria: r.FormValue("acceptance_criteria"),
		Notes:              r.FormValue("notes"),
		Status:             types.StatusOpen,
		Priority:           priority,
		IssueType:          types.IssueType(r.FormValue("type")),
		Assignee:           r.FormValue("assignee"),
	}

	// Validate
	if issue.Title == "" {
		h.renderError(w, r, "Title is required", nil)
		return
	}

	// Create issue
	if err := h.server.store.CreateIssue(ctx, issue, h.server.actor); err != nil {
		h.renderError(w, r, "Failed to create issue", err)
		return
	}

	// Redirect to detail page
	http.Redirect(w, r, "/"+issue.ID, http.StatusSeeOther)
}

// Helper functions

func (h *Handlers) render(w http.ResponseWriter, tmpl string, data PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.server.templates.ExecuteTemplate(w, tmpl, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		if h.server.debug {
			fmt.Fprintf(w, "\nDebug: %v", err)
		}
	}
}

func (h *Handlers) renderError(w http.ResponseWriter, r *http.Request, message string, err error) {
	errorMsg := message
	if err != nil && h.server.debug {
		errorMsg = fmt.Sprintf("%s: %v", message, err)
	}

	pageData := PageData{
		Title:       "Error",
		CurrentPath: r.URL.Path,
		Error:       errorMsg,
	}

	h.render(w, "error.html", pageData)
}

func parseListFilters(r *http.Request) ListFilters {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	return ListFilters{
		Status:         r.URL.Query().Get("status"),
		Priority:       r.URL.Query().Get("priority"),
		Type:           r.URL.Query().Get("type"),
		Assignee:       r.URL.Query().Get("assignee"),
		ParentID:       r.URL.Query().Get("parent"),
		ChildID:        r.URL.Query().Get("child"),
		DiscoveredFrom: r.URL.Query().Get("discovered"),
		Sort:           r.URL.Query().Get("sort"),
		Order:          r.URL.Query().Get("order"),
		Page:           page,
	}
}

func isValidIssueID(id string) bool {
	// Pattern matches: bd-42, test-web-1, my-prefix-100, etc.
	// Must start with lowercase letter, then allow letters/digits/hyphens, then hyphen and number
	matched, _ := regexp.MatchString(`^[a-z][a-z0-9-]+-[0-9]+$`, id)
	return matched
}

func (h *Handlers) filterByDependencies(ctx context.Context, issues []*types.Issue, filters ListFilters) []*types.Issue {
	// If no dependency filters, return as-is
	if filters.ParentID == "" && filters.ChildID == "" && filters.DiscoveredFrom == "" {
		return issues
	}

	filtered := make([]*types.Issue, 0)

	for _, issue := range issues {
		include := true

		// Check parent filter
		if filters.ParentID != "" {
			deps, _ := h.server.store.GetDependencyRecords(ctx, issue.ID)
			found := false
			for _, dep := range deps {
				if dep.DependsOnID == filters.ParentID &&
					(dep.Type == types.DepBlocks || dep.Type == types.DepParentChild) {
					found = true
					break
				}
			}
			include = include && found
		}

		// Check child filter (reverse lookup)
		if filters.ChildID != "" {
			deps, _ := h.server.store.GetDependencyRecords(ctx, filters.ChildID)
			found := false
			for _, dep := range deps {
				if dep.DependsOnID == issue.ID &&
					(dep.Type == types.DepBlocks || dep.Type == types.DepParentChild) {
					found = true
					break
				}
			}
			include = include && found
		}

		// Check discovered-from filter
		if filters.DiscoveredFrom != "" {
			deps, _ := h.server.store.GetDependencyRecords(ctx, issue.ID)
			found := false
			for _, dep := range deps {
				if dep.DependsOnID == filters.DiscoveredFrom && dep.Type == types.DepDiscoveredFrom {
					found = true
					break
				}
			}
			include = include && found
		}

		if include {
			filtered = append(filtered, issue)
		}
	}

	return filtered
}

func sortIssues(issues []*types.Issue, sortBy, order string) []*types.Issue {
	if sortBy == "" {
		sortBy = "id"
	}
	if order == "" {
		order = "desc"
	}

	// Sort in place using a simple bubble sort (good enough for small lists)
	// For production, use sort.Slice
	result := make([]*types.Issue, len(issues))
	copy(result, issues)

	// Use Go's built-in sort for better performance
	switch sortBy {
	case "title":
		if order == "asc" {
			for i := 0; i < len(result); i++ {
				for j := i + 1; j < len(result); j++ {
					if strings.ToLower(result[i].Title) > strings.ToLower(result[j].Title) {
						result[i], result[j] = result[j], result[i]
					}
				}
			}
		} else {
			for i := 0; i < len(result); i++ {
				for j := i + 1; j < len(result); j++ {
					if strings.ToLower(result[i].Title) < strings.ToLower(result[j].Title) {
						result[i], result[j] = result[j], result[i]
					}
				}
			}
		}
	case "updated":
		if order == "asc" {
			for i := 0; i < len(result); i++ {
				for j := i + 1; j < len(result); j++ {
					if result[i].UpdatedAt.After(result[j].UpdatedAt) {
						result[i], result[j] = result[j], result[i]
					}
				}
			}
		} else {
			for i := 0; i < len(result); i++ {
				for j := i + 1; j < len(result); j++ {
					if result[i].UpdatedAt.Before(result[j].UpdatedAt) {
						result[i], result[j] = result[j], result[i]
					}
				}
			}
		}
	default: // id
		if order == "asc" {
			for i := 0; i < len(result); i++ {
				for j := i + 1; j < len(result); j++ {
					if result[i].ID > result[j].ID {
						result[i], result[j] = result[j], result[i]
					}
				}
			}
		} else {
			for i := 0; i < len(result); i++ {
				for j := i + 1; j < len(result); j++ {
					if result[i].ID < result[j].ID {
						result[i], result[j] = result[j], result[i]
					}
				}
			}
		}
	}

	return result
}
