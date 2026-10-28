package personalagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const dashboardStartupTimeout = 15 * time.Second
const dashboardStartupPollInterval = 100 * time.Millisecond
const dashboardHealthRequestTimeout = 500 * time.Millisecond
const dashboardRequestTimeout = 35 * time.Second
const dashboardResponseLimitBytes = 2 << 20

type DashboardOverview struct {
	KPI json.RawMessage `json:"kpi"`
}

type DashboardOverviewSource interface {
	Overview(context.Context) (DashboardOverview, error)
}

type dashboardProcess struct {
	baseURL    string
	cancel     context.CancelFunc
	done       chan struct{}
	ready      chan struct{}
	startupErr error
}

type DashboardBridge struct {
	workspace string
	client    *http.Client
	mu        sync.Mutex
	baseURL   string
	process   *dashboardProcess
}

func NewDashboardBridge(workspace string) *DashboardBridge {
	return &DashboardBridge{
		workspace: workspace,
		client: &http.Client{
			Timeout: dashboardRequestTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (b *DashboardBridge) Overview(ctx context.Context) (DashboardOverview, error) {
	baseURL, err := b.ensureRunning(ctx)
	if err != nil {
		return DashboardOverview{}, err
	}
	data, err := b.getJSON(ctx, baseURL+"/api/v1/overview/kpi")
	if err != nil {
		return DashboardOverview{}, err
	}
	return DashboardOverview{KPI: data}, nil
}

func (b *DashboardBridge) ensureRunning(ctx context.Context) (string, error) {
	b.mu.Lock()
	if b.baseURL != "" && b.process == nil {
		baseURL := b.baseURL
		b.mu.Unlock()
		return baseURL, nil
	}
	if b.process != nil {
		process := b.process
		select {
		case <-process.done:
			b.process = nil
			b.baseURL = ""
		default:
			b.mu.Unlock()
			select {
			case <-process.ready:
				if process.startupErr != nil {
					return "", process.startupErr
				}
				return process.baseURL, nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
	}

	backendDirectory := filepath.Join(b.workspace, "app", "backend")
	uvicorn := dashboardUvicornPath(b.workspace)
	if info, err := os.Stat(filepath.Join(backendDirectory, "app", "main.py")); err != nil || info.IsDir() {
		b.mu.Unlock()
		return "", errors.New("Cockpit Dashboard backend is not installed in the workspace")
	}
	if info, err := os.Stat(uvicorn); err != nil || info.IsDir() {
		b.mu.Unlock()
		return "", errors.New("Cockpit Dashboard Python environment is not installed in the workspace")
	}

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		b.mu.Unlock()
		return "", fmt.Errorf("reserve local Dashboard API port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		b.mu.Unlock()
		return "", fmt.Errorf("release reserved Dashboard API port: %w", err)
	}

	processContext, cancel := context.WithCancel(context.Background())
	command := exec.CommandContext(
		processContext,
		uvicorn,
		"app.main:app",
		"--host",
		"127.0.0.1",
		"--port",
		strconv.Itoa(port),
	)
	command.Dir = backendDirectory
	command.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1")
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Start(); err != nil {
		cancel()
		b.mu.Unlock()
		return "", fmt.Errorf("start local Dashboard API: %w", err)
	}

	process := &dashboardProcess{
		baseURL: "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)),
		cancel:  cancel,
		done:    make(chan struct{}),
		ready:   make(chan struct{}),
	}
	b.process = process
	b.baseURL = process.baseURL
	go func() {
		_ = command.Wait()
		close(process.done)
	}()
	b.mu.Unlock()

	process.startupErr = b.waitUntilReady(ctx, process)
	close(process.ready)
	if process.startupErr != nil {
		process.cancel()
		select {
		case <-process.done:
		case <-time.After(dashboardHealthRequestTimeout):
		}
		b.mu.Lock()
		if b.process == process {
			b.process = nil
			b.baseURL = ""
		}
		b.mu.Unlock()
		return "", process.startupErr
	}
	return process.baseURL, nil
}

func (b *DashboardBridge) waitUntilReady(ctx context.Context, process *dashboardProcess) error {
	startupContext, cancel := context.WithTimeout(ctx, dashboardStartupTimeout)
	defer cancel()
	ticker := time.NewTicker(dashboardStartupPollInterval)
	defer ticker.Stop()

	for {
		requestContext, cancelRequest := context.WithTimeout(startupContext, dashboardHealthRequestTimeout)
		request, err := http.NewRequestWithContext(requestContext, http.MethodGet, process.baseURL+"/health", nil)
		if err == nil {
			response, requestErr := b.client.Do(request)
			if response != nil {
				_ = response.Body.Close()
				if response.StatusCode == http.StatusOK {
					cancelRequest()
					return nil
				}
				err = fmt.Errorf("Dashboard health check returned %s", response.Status)
			} else {
				err = requestErr
			}
		}
		cancelRequest()
		select {
		case <-process.done:
			return fmt.Errorf("Dashboard API exited before becoming ready: %w", err)
		case <-startupContext.Done():
			return fmt.Errorf("wait for Dashboard API readiness: %w", startupContext.Err())
		case <-ticker.C:
		}
	}
}

func (b *DashboardBridge) getJSON(ctx context.Context, url string) (json.RawMessage, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create Dashboard API request: %w", err)
	}
	response, err := b.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request Dashboard overview: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, dashboardResponseLimitBytes))
	if err != nil {
		return nil, fmt.Errorf("read Dashboard overview: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Dashboard overview returned %s", response.Status)
	}
	if !json.Valid(data) {
		return nil, errors.New("Dashboard overview returned invalid JSON")
	}
	return json.RawMessage(data), nil
}

func (b *DashboardBridge) Close(ctx context.Context) error {
	b.mu.Lock()
	process := b.process
	b.process = nil
	if process != nil {
		b.baseURL = ""
	}
	b.mu.Unlock()
	if process == nil {
		return nil
	}

	process.cancel()
	select {
	case <-process.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func dashboardUvicornPath(workspace string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(workspace, ".venv", "Scripts", "uvicorn.exe")
	}
	return filepath.Join(workspace, ".venv", "bin", "uvicorn")
}
