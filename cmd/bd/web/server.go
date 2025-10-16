package web

import (
	"embed"
	"html/template"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/steveyegge/beads/internal/storage"
)

//go:embed templates/*.html static/*
var embeddedFS embed.FS

// Server represents the web server
type Server struct {
	store     storage.Storage
	actor     string
	debug     bool
	templates *template.Template
	handlers  *Handlers
}

// NewServer creates a new web server instance
func NewServer(store storage.Storage, actor string, debug bool) *Server {
	// Parse templates
	tmpl, err := template.ParseFS(embeddedFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	s := &Server{
		store:     store,
		actor:     actor,
		debug:     debug,
		templates: tmpl,
	}

	// Initialize handlers
	s.handlers = NewHandlers(s)

	return s
}

// Router returns the configured chi router
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	// Middleware
	if s.debug {
		r.Use(middleware.Logger)
	}
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// Static files
	r.Handle("/static/*", http.FileServer(http.FS(embeddedFS)))

	// Page routes
	r.Get("/", s.handlers.RedirectToIssues)
	r.Get("/issues", s.handlers.ListIssues)
	r.Get("/issues/new", s.handlers.NewIssue)
	r.Post("/issues", s.handlers.CreateIssue)

	// Issue detail by ID (must come after specific routes)
	// Pattern matches: bd-42, test-web-1, my-prefix-100, etc.
	r.Get("/{issueID:[a-z][a-z0-9-]+-[0-9]+}", s.handlers.IssueDetail)

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/issues", s.handlers.APIListIssues)
		r.Get("/issues/{id}", s.handlers.APIGetIssue)
		r.Post("/issues", s.handlers.APICreateIssue)
		r.Put("/issues/{id}", s.handlers.APIUpdateIssue)
		r.Post("/issues/{id}/deps", s.handlers.APIAddDependency)
		r.Delete("/issues/{id}/deps", s.handlers.APIRemoveDependency)
		r.Get("/issues/search", s.handlers.APISearchIssues)
	})

	return r
}
