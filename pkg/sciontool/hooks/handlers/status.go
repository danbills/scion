/*
Copyright 2025 The Scion Authors.
*/

// Package handlers provides hook handler implementations.
package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ptone/scion-agent/pkg/sciontool/hooks"
)

// StatusHandler manages agent status in a JSON file.
// It replicates the functionality of scion_tool.py's update_status function.
type StatusHandler struct {
	// StatusPath is the path to the agent-info.json file.
	StatusPath string

	mu sync.Mutex
}

// NewStatusHandler creates a new status handler.
func NewStatusHandler() *StatusHandler {
	home := os.Getenv("HOME")
	if home == "" {
		home = "/home/scion"
	}
	return &StatusHandler{
		StatusPath: filepath.Join(home, "agent-info.json"),
	}
}

// Handle processes an event and updates the agent status.
func (h *StatusHandler) Handle(event *hooks.Event) error {
	state := h.eventToState(event)
	if state == "" {
		return nil // Event doesn't trigger a state change
	}

	// Update operational status
	if err := h.UpdateStatus(state, false); err != nil {
		return err
	}

	// Claude-specific: ExitPlanMode asks user to approve plan
	if event.Dialect == "claude" && event.Name == hooks.EventToolStart && event.Data.ToolName == "ExitPlanMode" {
		return h.UpdateStatus(hooks.StateWaitingForInput, true)
	}

	// Claude-specific: AskUserQuestion maintains WAITING_FOR_INPUT that was
	// set by a prior "sciontool status ask_user" call (which runs in a Bash
	// tool whose PostToolUse could otherwise clear it).
	if event.Dialect == "claude" && event.Name == hooks.EventToolStart && event.Data.ToolName == "AskUserQuestion" {
		return h.UpdateStatus(hooks.StateWaitingForInput, true)
	}

	// New work events (new prompt, new agent turn, new session) indicate the
	// agent is starting fresh work. Clear any transient session status,
	// including COMPLETED (the previous task is done, new one is starting).
	if isNewWorkEvent(event.Name) {
		return h.ClearSessionStatus()
	}

	// Tool-start events indicate the agent is actively working within the
	// current task. Clear WAITING_FOR_INPUT (user has responded) but preserve
	// COMPLETED (tools may fire after task_completed as part of wrap-up).
	if event.Name == hooks.EventToolStart {
		return h.ClearWaitingStatus()
	}

	return nil
}

// UpdateStatus writes the status to the agent-info.json file atomically.
func (h *StatusHandler) UpdateStatus(status hooks.AgentState, sessionStatus bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Read existing data preserving all fields
	info := h.readAgentInfoMap()

	// Update the appropriate field
	if sessionStatus {
		if status == "" {
			delete(info, "sessionStatus")
		} else {
			info["sessionStatus"] = string(status)
		}
	} else {
		if status == "" {
			delete(info, "status")
		} else {
			info["status"] = string(status)
		}
	}

	return h.writeAgentInfoLocked(info)
}

// readAgentInfoMap reads agent-info.json into a generic map, preserving all fields.
// Caller must hold h.mu.
func (h *StatusHandler) readAgentInfoMap() map[string]interface{} {
	info := make(map[string]interface{})
	if data, err := os.ReadFile(h.StatusPath); err == nil {
		_ = json.Unmarshal(data, &info)
	}
	return info
}

// writeAgentInfoLocked writes the agent info map to disk atomically.
// Caller must hold h.mu.
func (h *StatusHandler) writeAgentInfoLocked(info map[string]interface{}) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling status: %w", err)
	}

	dir := filepath.Dir(h.StatusPath)
	tmpFile, err := os.CreateTemp(dir, "agent-info-*.json")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing temp file: %w", err)
	}
	tmpFile.Close()

	if err := os.Rename(tmpPath, h.StatusPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("atomic rename: %w", err)
	}

	return nil
}

// ClearWaitingStatus clears the sessionStatus if it is currently WAITING_FOR_INPUT.
// This is a no-op if sessionStatus is any other value (e.g., COMPLETED).
// Used for tool-start events where the agent is still working on the same task.
func (h *StatusHandler) ClearWaitingStatus() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	info := h.readAgentInfoMap()

	ss, _ := info["sessionStatus"].(string)
	if ss != string(hooks.StateWaitingForInput) {
		return nil // Not waiting, nothing to clear
	}

	delete(info, "sessionStatus")
	return h.writeAgentInfoLocked(info)
}

// ClearSessionStatus clears the sessionStatus if it is a transient state
// (WAITING_FOR_INPUT or COMPLETED). This is called when new work begins
// (new prompt, new agent turn, new session) to reset the session status.
func (h *StatusHandler) ClearSessionStatus() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	info := h.readAgentInfoMap()

	ss, _ := info["sessionStatus"].(string)
	switch ss {
	case string(hooks.StateWaitingForInput), string(hooks.StateCompleted):
		delete(info, "sessionStatus")
		return h.writeAgentInfoLocked(info)
	default:
		return nil // Nothing to clear
	}
}

// isNewWorkEvent returns true for events that indicate new work is starting.
// These events clear all transient session status (both WAITING_FOR_INPUT and
// COMPLETED), since the agent is beginning a new task or turn.
func isNewWorkEvent(name string) bool {
	switch name {
	case hooks.EventPromptSubmit, hooks.EventAgentStart, hooks.EventSessionStart:
		return true
	}
	return false
}

// eventToState maps normalized events to agent states.
func (h *StatusHandler) eventToState(event *hooks.Event) hooks.AgentState {
	switch event.Name {
	case hooks.EventSessionStart:
		return hooks.StateStarting

	case hooks.EventPreStart:
		return hooks.StateInitializing

	case hooks.EventPostStart:
		return hooks.StateIdle

	case hooks.EventPreStop:
		return hooks.StateShuttingDown

	case hooks.EventPromptSubmit, hooks.EventAgentStart:
		return hooks.StateThinking

	case hooks.EventModelStart:
		return hooks.StateThinking

	case hooks.EventModelEnd:
		return hooks.StateIdle

	case hooks.EventToolStart:
		// Include tool name in state if available
		if event.Data.ToolName != "" {
			// Return a dynamic state - caller should handle formatting
			return hooks.StateExecuting
		}
		return hooks.StateExecuting

	case hooks.EventToolEnd, hooks.EventAgentEnd:
		return hooks.StateIdle

	case hooks.EventNotification:
		return hooks.StateWaitingForInput

	case hooks.EventSessionEnd:
		return hooks.StateExited

	default:
		return "" // No state change
	}
}

// GetFormattedState returns the state with tool name if applicable.
func (h *StatusHandler) GetFormattedState(event *hooks.Event) string {
	state := h.eventToState(event)
	if state == hooks.StateExecuting && event.Data.ToolName != "" {
		return fmt.Sprintf("%s (%s)", state, event.Data.ToolName)
	}
	return string(state)
}
