package infra

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	buildBinaryOnce sync.Once
	builtBinaryPath string
	buildBinaryErr  error
)

// EnsureBinary locates or compiles the GoRoute gateway binary for E2E execution.
// If GOROUTE_BIN environment variable is set and points to an executable, it is used.
// Otherwise, it compiles cmd/goroute once into a shared temporary executable.
func EnsureBinary() (string, error) {
	buildBinaryOnce.Do(func() {
		// 1. Check environment variable override
		if envBin := os.Getenv("GOROUTE_BIN"); envBin != "" {
			if info, err := os.Stat(envBin); err == nil && !info.IsDir() {
				builtBinaryPath = envBin
				return
			}
		}

		// 2. Locate GoRoute project root containing go.mod
		projRoot, err := findProjectRoot()
		if err != nil {
			buildBinaryErr = fmt.Errorf("failed to locate project root: %w", err)
			return
		}

		// 3. Compile GoRoute binary into temp file
		tmpFile, err := os.CreateTemp("", "goroute-e2e-bin-*")
		if err != nil {
			buildBinaryErr = fmt.Errorf("failed to create temp binary file: %w", err)
			return
		}
		binPath := tmpFile.Name()
		_ = tmpFile.Close()

		buildCmd := exec.Command("go", "build", "-o", binPath, "./cmd/goroute")
		buildCmd.Dir = projRoot
		var buildErr bytes.Buffer
		buildCmd.Stderr = &buildErr

		if err := buildCmd.Run(); err != nil {
			_ = os.Remove(binPath)
			buildBinaryErr = fmt.Errorf("failed to compile goroute: %w (stderr: %s)", err, buildErr.String())
			return
		}

		builtBinaryPath = binPath
	})

	return builtBinaryPath, buildBinaryErr
}

// CleanupBuiltBinary removes the compiled temporary binary if one was created.
func CleanupBuiltBinary() {
	if builtBinaryPath != "" && os.Getenv("GOROUTE_BIN") == "" {
		_ = os.Remove(builtBinaryPath)
	}
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found in hierarchy")
}

// GetFreePort dynamically allocates an available localhost TCP port.
func GetFreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("failed to allocate free port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		return 0, err
	}
	return port, nil
}

// ConfigTemplate defines options for generating an ephemeral config.yaml for testing.
type ConfigTemplate struct {
	Port            int
	Host            string
	EnableDashboard bool
	Providers       []map[string]interface{}
	Combos          []map[string]interface{}
	Keys            []map[string]interface{}
}

// HarnessOption configures a TestHarness instance.
type HarnessOption func(*TestHarness)

// WithPort sets an explicit port for the harness daemon.
func WithPort(port int) HarnessOption {
	return func(h *TestHarness) {
		h.port = port
	}
}

// WithDashboard enables or disables the embedded web dashboard.
func WithDashboard(enabled bool) HarnessOption {
	return func(h *TestHarness) {
		h.enableDashboard = enabled
	}
}

// WithCustomConfig sets raw YAML config string to write into config.yaml.
func WithCustomConfig(yamlConfig string) HarnessOption {
	return func(h *TestHarness) {
		h.customConfigYAML = yamlConfig
	}
}

// WithCustomEnv sets custom environment variables on the daemon process.
func WithCustomEnv(key, val string) HarnessOption {
	return func(h *TestHarness) {
		h.customEnv[key] = val
	}
}

// TestHarness manages the lifecycle of an isolated, ephemeral GoRoute daemon instance.
type TestHarness struct {
	t                *testing.T
	rootDir          string
	gorouteDir       string
	cfgPath          string
	dbPath           string
	vaultPath        string
	port             int
	enableDashboard  bool
	customConfigYAML string
	customEnv        map[string]string

	cmd      *exec.Cmd
	stdout   bytes.Buffer
	stderr   bytes.Buffer
	client   *Client
	verifier *SQLiteVerifier

	mu        sync.Mutex
	running   bool
	startedAt time.Time
}

// NewHarness creates a new test harness with an isolated temporary HOME directory.
func NewHarness(t *testing.T, opts ...HarnessOption) *TestHarness {
	rootDir, err := os.MkdirTemp("", "goroute-test-harness-*")
	if err != nil {
		t.Fatalf("failed to create harness temp directory: %v", err)
	}

	gorouteDir := filepath.Join(rootDir, ".goroute")
	if err := os.MkdirAll(gorouteDir, 0700); err != nil {
		t.Fatalf("failed to create .goroute directory: %v", err)
	}

	h := &TestHarness{
		t:          t,
		rootDir:    rootDir,
		gorouteDir: gorouteDir,
		cfgPath:    filepath.Join(gorouteDir, "config.yaml"),
		dbPath:     filepath.Join(gorouteDir, "goroute.db"),
		vaultPath:  filepath.Join(gorouteDir, "vault.json"),
		customEnv:  make(map[string]string),
	}

	for _, opt := range opts {
		opt(h)
	}

	// Register teardown on test completion
	t.Cleanup(func() {
		h.Teardown()
	})

	return h
}

// Start compiles the binary (if needed), writes config, and starts the daemon process.
func (h *TestHarness) Start() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return fmt.Errorf("daemon is already running")
	}

	binPath, err := EnsureBinary()
	if err != nil {
		return fmt.Errorf("failed to prepare goroute binary: %w", err)
	}

	// Allocate port if not specified
	if h.port == 0 {
		freePort, err := GetFreePort()
		if err != nil {
			return err
		}
		h.port = freePort
	}

	// Write configuration file
	if err := h.writeConfigFile(); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Build process arguments
	args := []string{"serve", "-c", h.cfgPath, "-p", strconv.Itoa(h.port)}
	if h.enableDashboard {
		args = append(args, "--dashboard")
	} else {
		args = append(args, "--no-dashboard")
	}

	cmd := exec.Command(binPath, args...)
	cmd.Stdout = &h.stdout
	cmd.Stderr = &h.stderr

	// Isolate process environment to temporary root dir
	env := os.Environ()
	env = append(env,
		"HOME="+h.rootDir,
		"USERPROFILE="+h.rootDir,
		fmt.Sprintf("GOROUTE_PORT=%d", h.port),
	)
	if h.enableDashboard {
		env = append(env, "GOROUTE_ENABLE_DASHBOARD=1")
	} else {
		env = append(env, "GOROUTE_NO_DASHBOARD=1")
	}
	for k, v := range h.customEnv {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon process: %w (stderr: %s)", err, h.stderr.String())
	}

	h.cmd = cmd
	h.running = true
	h.startedAt = time.Now()

	// Initialize Client pointing to ephemeral instance
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", h.port)
	h.client = NewClient(baseURL)

	// Wait for health endpoint to respond
	if err := h.waitForHealthy(10 * time.Second); err != nil {
		_ = h.cmd.Process.Kill()
		h.running = false
		return fmt.Errorf("daemon failed to reach healthy state: %w\nStdout:\n%s\nStderr:\n%s", err, h.stdout.String(), h.stderr.String())
	}

	return nil
}

func (h *TestHarness) writeConfigFile() error {
	if h.customConfigYAML != "" {
		return os.WriteFile(h.cfgPath, []byte(h.customConfigYAML), 0600)
	}

	// Write minimal valid default configuration
	cfgMap := map[string]interface{}{
		"server": map[string]interface{}{
			"host":              "127.0.0.1",
			"port":              h.port,
			"enable_dashboard":  h.enableDashboard,
			"disable_dashboard": !h.enableDashboard,
		},
		"providers": []interface{}{},
		"combos":    []interface{}{},
		"keys":      []interface{}{},
	}

	data, err := yaml.Marshal(cfgMap)
	if err != nil {
		return err
	}
	return os.WriteFile(h.cfgPath, data, 0600)
}

// UpdateConfig writes new YAML configuration and optionally calls reload.
func (h *TestHarness) UpdateConfig(yamlContent string, reload bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := os.WriteFile(h.cfgPath, []byte(yamlContent), 0600); err != nil {
		return err
	}

	if reload && h.running {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _, err := h.client.Reload(ctx)
		return err
	}
	return nil
}

// waitForHealthy polls /health until HTTP 200 is received or timeout expires.
func (h *TestHarness) waitForHealthy(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// Verify process hasn't exited prematurely
		if h.cmd != nil && h.cmd.ProcessState != nil && h.cmd.ProcessState.Exited() {
			return fmt.Errorf("process exited prematurely with code %d", h.cmd.ProcessState.ExitCode())
		}

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		code, _, err := h.client.Health(ctx)
		cancel()

		if err == nil && code == http.StatusOK {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("health check timed out after %v", timeout)
}

// Stop initiates graceful shutdown, waiting up to 5 seconds before force-killing.
func (h *TestHarness) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running || h.cmd == nil || h.cmd.Process == nil {
		return nil
	}

	// Close SQLite verifier before killing process to avoid locked handles
	if h.verifier != nil {
		_ = h.verifier.Close()
		h.verifier = nil
	}

	// 1. Try graceful shutdown via API
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	_, _, _ = h.client.Shutdown(ctx)
	cancel()

	// 2. Wait up to 3 seconds for exit
	done := make(chan error, 1)
	go func() {
		done <- h.cmd.Wait()
	}()

	select {
	case <-done:
		h.running = false
		return nil
	case <-time.After(3 * time.Second):
		// 3. Fallback: Send SIGTERM or SIGKILL
		_ = h.cmd.Process.Signal(syscall.SIGTERM)
	}

	select {
	case <-done:
		h.running = false
		return nil
	case <-time.After(2 * time.Second):
		_ = h.cmd.Process.Kill()
		<-done
		h.running = false
		return nil
	}
}

// StopWithSignal sends a specific OS signal (e.g. SIGINT or SIGTERM) and asserts clean termination.
func (h *TestHarness) StopWithSignal(sig os.Signal, timeout time.Duration) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running || h.cmd == nil || h.cmd.Process == nil {
		return fmt.Errorf("daemon is not running")
	}

	if h.verifier != nil {
		_ = h.verifier.Close()
		h.verifier = nil
	}

	if err := h.cmd.Process.Signal(sig); err != nil {
		return fmt.Errorf("failed to signal process: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- h.cmd.Wait()
	}()

	select {
	case err := <-done:
		h.running = false
		// A signal termination exit code is expected
		_ = err
		return nil
	case <-time.After(timeout):
		_ = h.cmd.Process.Kill()
		<-done
		h.running = false
		return fmt.Errorf("process did not terminate within %v after signal %s", timeout, sig)
	}
}

// Teardown cleanly terminates the process and cleans up temporary test directory.
func (h *TestHarness) Teardown() {
	_ = h.Stop()

	if h.verifier != nil {
		_ = h.verifier.Close()
		h.verifier = nil
	}

	if os.Getenv("KEEP_TEST_DATA") != "1" && h.rootDir != "" {
		_ = os.RemoveAll(h.rootDir)
	}
}

// Client returns the preconfigured opaque Client.
func (h *TestHarness) Client() *Client {
	return h.client
}

// Verifier initializes (if not already opened) and returns the read-only SQLiteVerifier.
func (h *TestHarness) Verifier() (*SQLiteVerifier, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.verifier != nil {
		return h.verifier, nil
	}

	// Verify database file exists on disk
	if _, err := os.Stat(h.dbPath); err != nil {
		return nil, fmt.Errorf("sqlite database not found at %s: %w", h.dbPath, err)
	}

	v, err := NewSQLiteVerifier(h.dbPath)
	if err != nil {
		return nil, err
	}
	h.verifier = v
	return v, nil
}

// Port returns the assigned listening port.
func (h *TestHarness) Port() int {
	return h.port
}

// BaseURL returns the daemon base URL string.
func (h *TestHarness) BaseURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", h.port)
}

// DBPath returns path to the SQLite database file.
func (h *TestHarness) DBPath() string {
	return h.dbPath
}

// ConfigPath returns path to config.yaml.
func (h *TestHarness) ConfigPath() string {
	return h.cfgPath
}

// RootDir returns path to isolated temporary root directory.
func (h *TestHarness) RootDir() string {
	return h.rootDir
}

// Stdout returns captured standard output logs.
func (h *TestHarness) Stdout() string {
	return h.stdout.String()
}

// Stderr returns captured standard error logs.
func (h *TestHarness) Stderr() string {
	return h.stderr.String()
}

// IsRunning reports whether the daemon process is active.
func (h *TestHarness) IsRunning() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.running
}

// PID returns the daemon OS process ID if running, or 0.
func (h *TestHarness) PID() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cmd != nil && h.cmd.Process != nil {
		return h.cmd.Process.Pid
	}
	return 0
}
