package service

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

type ServiceResult struct {
	Success   bool      `json:"success"`
	Output    string    `json:"output"`
	Status    string    `json:"status"`
	Duration  string    `json:"duration"`
	Timestamp time.Time `json:"timestamp"`
}

type OxidizedManager struct {
	mu             sync.Mutex
	restartCommand string
	statusCommand  string
}

func NewOxidizedManager(restartCmd, statusCmd string) *OxidizedManager {
	if restartCmd == "" {
		restartCmd = "systemctl restart oxidized.service"
	}
	if statusCmd == "" {
		statusCmd = "systemctl is-active oxidized.service"
	}
	return &OxidizedManager{
		restartCommand: restartCmd,
		statusCommand:  statusCmd,
	}
}

// Restart executes the restart command with a timeout
func (m *OxidizedManager) Restart(ctx context.Context) ServiceResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	startTime := time.Now()

	// If timeout is not set on context, apply 20s timeout
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
	}

	cmdStr := m.restartCommand
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// If running on Windows during development, check if command is mock or systemctl
		if strings.Contains(cmdStr, "systemctl") {
			return ServiceResult{
				Success:   true,
				Output:    "[DEV MODE on Windows] Simulated restart of oxidized.service successfully.",
				Status:    "simulated_active",
				Duration:  time.Since(startTime).String(),
				Timestamp: time.Now(),
			}
		}
		cmd = exec.CommandContext(ctx, "cmd", "/C", cmdStr)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", cmdStr)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(startTime)

	output := strings.TrimSpace(stdout.String())
	errOut := strings.TrimSpace(stderr.String())
	if errOut != "" {
		if output != "" {
			output += "\n" + errOut
		} else {
			output = errOut
		}
	}

	status := m.checkStatusUnlocked(ctx)

	if err != nil {
		if output == "" {
			output = err.Error()
		}
		return ServiceResult{
			Success:   false,
			Output:    output,
			Status:    status,
			Duration:  duration.String(),
			Timestamp: time.Now(),
		}
	}

	if output == "" {
		output = fmt.Sprintf("Service command executed successfully: %s", cmdStr)
	}

	return ServiceResult{
		Success:   true,
		Output:    output,
		Status:    status,
		Duration:  duration.String(),
		Timestamp: time.Now(),
	}
}

// Status checks the current state of oxidized.service
func (m *OxidizedManager) Status(ctx context.Context) ServiceResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	startTime := time.Now()
	status := m.checkStatusUnlocked(ctx)

	return ServiceResult{
		Success:   status == "active" || status == "simulated_active",
		Status:    status,
		Duration:  time.Since(startTime).String(),
		Timestamp: time.Now(),
	}
}

func (m *OxidizedManager) checkStatusUnlocked(ctx context.Context) string {
	if runtime.GOOS == "windows" {
		return "simulated_active"
	}

	cmdStr := m.statusCommand
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	out, err := cmd.Output()
	result := strings.TrimSpace(string(out))

	if err != nil {
		if result != "" {
			return result
		}
		return "inactive"
	}

	if result == "" {
		return "active"
	}
	return result
}
