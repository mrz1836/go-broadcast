package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mrz1836/go-broadcast/internal/output"
)

// CLIResponse is the standard JSON envelope for all CRUD commands.
// It provides a consistent structure for AI agents and programmatic consumers.
type CLIResponse struct {
	Success bool        `json:"success"`
	Action  string      `json:"action"`          // "created", "updated", "deleted", "listed", "attached", "detached"
	Type    string      `json:"type"`            // "group", "target", "file_list", "directory_list", etc.
	Data    interface{} `json:"data"`            // result object or array
	Count   int         `json:"count,omitempty"` // for lists
	Error   string      `json:"error,omitempty"` // error message on failure
	Hint    string      `json:"hint,omitempty"`  // actionable suggestion on failure
}

// printResponse outputs a CLIResponse. When jsonOutput is true, it writes structured JSON
// to stdout. When false, it writes human-readable text to stdout.
func printResponse(resp CLIResponse, jsonOutput bool) error {
	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(resp)
	}

	// Human-readable output
	if resp.Success {
		output.Success(fmt.Sprintf("%s %s successfully", resp.Type, resp.Action))
	}
	return nil
}

// printErrorResponse outputs an error CLIResponse. When jsonOutput is true, it writes
// structured JSON to stdout. When false, it returns the error for Cobra to handle.
func printErrorResponse(entityType, action, errMsg, hint string, jsonOutput bool) error {
	if jsonOutput {
		resp := CLIResponse{
			Success: false,
			Action:  action,
			Type:    entityType,
			Error:   errMsg,
			Hint:    hint,
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(resp)
	}
	if hint != "" {
		return fmt.Errorf("%s: %s (hint: %s)", action, errMsg, hint) //nolint:err113 // user-facing CLI error
	}
	return fmt.Errorf("%s: %s", action, errMsg) //nolint:err113 // user-facing CLI error
}
