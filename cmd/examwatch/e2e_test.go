//go:build e2e

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/creack/pty"
)

var binPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "examwatch-e2e-")
	if err != nil {
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	binPath = filepath.Join(tmpDir, "examwatch")
	cmd := exec.Command("go", "build", "-tags", "e2e", "-o", binPath, ".")
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestHappyPath(t *testing.T) {
	tmpOut := t.TempDir()
	fixPath := filepath.Join(tmpOut, "fix.json")
	_ = os.WriteFile(fixPath, []byte(`[{"rtt": 10, "ip": "1.1.1.1", "pmset": "AC Power"}]`), 0644)

	cmd := exec.Command(binPath, "run", "--duration", "2s", "--interval", "500ms", "--out", tmpOut)
	cmd.Env = append(os.Environ(), "EXAMWATCH_FAKE_PROBES="+fixPath)
	
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	
	if err := cmd.Run(); err != nil {
		t.Fatalf("expected success, got %v\nOutput: %s", err, out.String())
	}
	
	if _, err := os.Stat(filepath.Join(tmpOut, "log.jsonl")); os.IsNotExist(err) {
		t.Error("log.jsonl not created")
	}
	if _, err := os.Stat(filepath.Join(tmpOut, "summary.json")); os.IsNotExist(err) {
		t.Error("summary.json not created")
	}
	if _, err := os.Stat(filepath.Join(tmpOut, "report.md")); os.IsNotExist(err) {
		t.Error("report.md not created")
	}
}

func TestSustainedLoss(t *testing.T) {
	tmpOut := t.TempDir()
	fixPath := filepath.Join(tmpOut, "fix.json")
	// 5 seconds of loss = FAIL
	_ = os.WriteFile(fixPath, []byte(`[{"rtt": 10, "loss": true, "ip": "1.1.1.1", "pmset": "AC Power"}]`), 0644)

	cmd := exec.Command(binPath, "run", "--duration", "8s", "--interval", "1s", "--out", tmpOut)
	cmd.Env = append(os.Environ(), "EXAMWATCH_FAKE_PROBES="+fixPath)
	
	if err := cmd.Run(); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	
	b, err := os.ReadFile(filepath.Join(tmpOut, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"status": "FAIL"`) || !strings.Contains(string(b), `"name": "SUSTAINED_LOSS"`) {
		t.Errorf("expected SUSTAINED_LOSS FAIL, got %s", string(b))
	}
}

func TestTTY(t *testing.T) {
	tmpOut := t.TempDir()
	fixPath := filepath.Join(tmpOut, "fix.json")
	_ = os.WriteFile(fixPath, []byte(`[{"rtt": 10, "ip": "1.1.1.1", "pmset": "AC Power"}]`), 0644)

	cmd := exec.Command(binPath, "run", "--duration", "1s", "--interval", "500ms", "--out", tmpOut)
	cmd.Env = append(os.Environ(), "EXAMWATCH_FAKE_PROBES="+fixPath)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ptmx.Close() }()

	buf := new(bytes.Buffer)
	go func() {
		_, _ = buf.ReadFrom(ptmx)
	}()

	_ = cmd.Wait()

	out := buf.String()
	if !strings.Contains(out, "\033[H") {
		t.Errorf("expected clear screen sequence in TTY mode, got %q", out)
	}
}
