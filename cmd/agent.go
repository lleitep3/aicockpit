package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/personalagent"
	"github.com/spf13/cobra"
)

func NewAgentCommand() *cobra.Command {
	var workspace string
	var port int
	var desktop bool
	command := &cobra.Command{
		Use:   "agent",
		Short: "Start the local Cockpit Personal Agent interface",
		Long: `Start the local Cockpit agent interface. The service binds only to 127.0.0.1.
Conversations are stored under ~/.cockpit/conversations. The workspace defaults
to ~/.cockpit/workspace; pass --workspace to select another directory.
Pass --desktop to open the Electron desktop window.`,
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			if port < 0 || port > 65535 {
				return fmt.Errorf("port must be between 0 and 65535")
			}
			if desktop {
				return runAgentDesktop(command.Context())
			}
			workspace = resolveAgentWorkspace(workspace)
			return runAgentServer(command.Context(), command, workspace, port)
		},
	}
	command.Flags().StringVar(&workspace, "workspace", "", "Workspace directory available to the agent (default: ~/.cockpit/workspace)")
	command.Flags().IntVar(&port, "port", 0, "Loopback TCP port (0 selects an available port)")
	command.Flags().BoolVar(&desktop, "desktop", false, "Open the Cockpit Personal Agent inside the Electron desktop shell")
	return command
}

func runAgentDesktop(ctx context.Context) error {
	desktopDir := "/home/lleite/projects/aicockpit/internal/personalagent/desktop"
	electronBin := filepath.Join(desktopDir, "node_modules", ".bin", "electron")
	if _, err := os.Stat(electronBin); err != nil {
		return fmt.Errorf("Electron binary not found at %s (run npm ci in %s)", electronBin, desktopDir)
	}
	cmd := exec.CommandContext(ctx, electronBin, "--no-sandbox", desktopDir)
	cmd.Dir = desktopDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func resolveAgentWorkspace(workspace string) string {
	if workspace != "" {
		return workspace
	}
	return filepath.Join(config.GetCockpitDir(), "workspace")
}

func runAgentServer(parent context.Context, command *cobra.Command, workspace string, port int) error {
	store, err := personalagent.NewStore(filepath.Join(config.GetCockpitDir(), "conversations"))
	if err != nil {
		return err
	}
	service, err := personalagent.NewService(store, personalagent.NewProviderRegistry(), workspace)
	if err != nil {
		return err
	}
	token, err := personalagent.NewID("session_")
	if err != nil {
		return err
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		return fmt.Errorf("listen on loopback: %w", err)
	}
	defer listener.Close()
	host := listener.Addr().String()
	server := &http.Server{
		Handler:           personalagent.NewHTTPHandler(service, token, host),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()
	shutdownDone := make(chan struct{})
	var shutdownErr error
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		httpCtx, cancelHTTP := context.WithTimeout(context.Background(), 5*time.Second)
		httpErr := server.Shutdown(httpCtx)
		cancelHTTP()
		serviceCtx, cancelService := context.WithTimeout(context.Background(), 8*time.Second)
		serviceErr := service.Shutdown(serviceCtx)
		cancelService()
		shutdownErr = errors.Join(httpErr, serviceErr)
	}()
	_, _ = fmt.Fprintf(command.OutOrStdout(), "Cockpit Personal Agent\nWorkspace: %s\nOpen: http://%s\nPress Ctrl+C to stop.\n", service.Workspace(), host)
	serveErr := server.Serve(listener)
	if !errors.Is(serveErr, http.ErrServerClosed) {
		stop()
	}
	<-shutdownDone
	if shutdownErr != nil {
		return fmt.Errorf("shutdown local agent: %w", shutdownErr)
	}
	if errors.Is(serveErr, http.ErrServerClosed) {
		return nil
	}
	return serveErr
}
