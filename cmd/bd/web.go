package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/steveyegge/beads/cmd/bd/web"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start web interface for managing issues",
	Long: `Start a local web server for managing issues through a browser interface.

The web interface provides:
- List view with filtering and sorting
- Issue detail pages
- Inline editing
- Dependency management
- Issue creation`,
	Run: runWebServer,
}

var (
	webPort  int
	webBind  string
	webOpen  bool
	webDebug bool
)

func init() {
	webCmd.Flags().IntVarP(&webPort, "port", "p", 8080, "Port to listen on")
	webCmd.Flags().StringVarP(&webBind, "bind", "b", "127.0.0.1", "Address to bind to")
	webCmd.Flags().BoolVarP(&webOpen, "open", "o", false, "Open browser automatically")
	webCmd.Flags().BoolVar(&webDebug, "debug", false, "Enable debug logging")
	rootCmd.AddCommand(webCmd)
}

func runWebServer(cmd *cobra.Command, args []string) {
	// Create server with the storage that was initialized in PersistentPreRun
	srv := web.NewServer(store, actor, webDebug)

	// Setup address
	addr := net.JoinHostPort(webBind, fmt.Sprintf("%d", webPort))

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      srv.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		green := color.New(color.FgGreen).SprintFunc()
		cyan := color.New(color.FgCyan).SprintFunc()

		fmt.Printf("\n%s Beads web interface starting...\n", green("✓"))
		fmt.Printf("  %s http://%s\n", cyan("→"), addr)
		fmt.Printf("  Press Ctrl+C to stop\n\n")

		// Open browser if requested
		if webOpen {
			openBrowser(fmt.Sprintf("http://%s", addr))
		}

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt
	<-stop

	// Shutdown gracefully
	yellow := color.New(color.FgYellow).SprintFunc()
	fmt.Printf("\n%s Shutting down server...\n", yellow("⚠"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	fmt.Println("Server stopped")
}

// openBrowser opens the default browser to the given URL
func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		fmt.Printf("Please open %s in your browser\n", url)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
		fmt.Printf("Please open %s manually\n", url)
	}
}
