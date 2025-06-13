package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Constants for log processing
const (
	maxDisplayLength     = 100
	maxSummaryLength     = 60
	maxInputLength       = 80
	maxFirstLineLength   = 30
	shortResultThreshold = 60
	longOutputThreshold  = 5
)

// LogProcessor processes Claude execution logs for human-readable display
type LogProcessor struct{}

// NewLogProcessor creates a new log processor
func NewLogProcessor() *LogProcessor {
	return &LogProcessor{}
}

// ProcessExecution processes an execution's logs and returns formatted output
func (lp *LogProcessor) ProcessExecution(metadata *ExecutionMetadata, execMgr *ExecutionManager) (string, error) {
	// Load raw log using the unified file finding logic
	logFile := FindLogFileByExecutionID(execMgr.GetLogDir(), metadata.StartTime, metadata.ExecutionID)

	logEntries, err := lp.loadJSONLog(logFile)
	if err != nil {
		return "", fmt.Errorf("failed to load log: %w", err)
	}

	// Parse log entries
	conversations := lp.extractConversations(logEntries)
	toolUses := lp.extractToolUses(logEntries)
	results := lp.extractResults(logEntries)
	operationFlow := lp.extractOperationFlow(logEntries)

	// Format output
	formatted := lp.formatExecution(metadata, conversations, toolUses, results, operationFlow)
	return formatted, nil
}

// JSONLogEntry represents a single log entry
type JSONLogEntry struct {
	Type      string                 `json:"type"`
	Subtype   string                 `json:"subtype,omitempty"`
	Message   map[string]interface{} `json:"message,omitempty"`
	Result    string                 `json:"result,omitempty"`
	CostUSD   float64                `json:"cost_usd,omitempty"`
	Duration  int64                  `json:"duration_ms,omitempty"`
	Model     string                 `json:"model,omitempty"`
	Timestamp string                 `json:"timestamp,omitempty"`
	Raw       map[string]interface{} `json:"-"` // Store raw data
}

// Conversation represents a parsed conversation
type Conversation struct {
	Role    string `json:"role"` // "assistant" or "user"
	Content string `json:"content"`
	Type    string `json:"type"` // "text", "tool_use", "tool_result"
}

// ToolUse represents a tool usage
type ToolUse struct {
	Name    string `json:"name"`
	Input   string `json:"input"`
	Output  string `json:"output"`
	Success bool   `json:"success"`
}

// Result represents the final result
type Result struct {
	Success  bool    `json:"success"`
	Message  string  `json:"message"`
	CostUSD  float64 `json:"cost_usd"`
	Duration int64   `json:"duration_ms"`
}

// OperationStep represents a single step in the operation flow
type OperationStep struct {
	StepNumber int    `json:"step_number"`
	Type       string `json:"type"`  // "assistant_message", "tool_use", "tool_result", "system"
	Actor      string `json:"actor"` // "assistant", "user", "system"
	Content    string `json:"content"`
	Details    string `json:"details,omitempty"`
	Success    bool   `json:"success"`
	Timestamp  string `json:"timestamp,omitempty"`
}

// loadJSONLog loads and parses the JSON log file
func (lp *LogProcessor) loadJSONLog(logFile string) ([]JSONLogEntry, error) {
	file, err := os.Open(logFile)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Warning: failed to close log file: %v\n", err)
		}
	}()

	var entries []JSONLogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry JSONLogEntry
		var raw map[string]interface{}

		// Parse into raw map first
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue // Skip invalid JSON
		}

		// Parse into structured entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		entry.Raw = raw
		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}

// extractConversations extracts conversation messages
func (lp *LogProcessor) extractConversations(entries []JSONLogEntry) []Conversation {
	var conversations []Conversation

	for _, entry := range entries {
		if entry.Type == "assistant" && entry.Message != nil {
			if content, ok := entry.Message["content"].([]interface{}); ok {
				for _, item := range content {
					if contentItem, ok := item.(map[string]interface{}); ok {
						switch contentItem["type"] {
						case "text":
							if text, ok := contentItem["text"].(string); ok {
								conversations = append(conversations, Conversation{
									Role:    "assistant",
									Content: text,
									Type:    "text",
								})
							}
						case "tool_use":
							if name, ok := contentItem["name"].(string); ok {
								input := "No input"
								if inputData, ok := contentItem["input"].(map[string]interface{}); ok {
									if inputStr, err := json.MarshalIndent(inputData, "", "  "); err == nil {
										input = string(inputStr)
									}
								}
								conversations = append(conversations, Conversation{
									Role:    "assistant",
									Content: fmt.Sprintf("Using tool: %s\nInput: %s", name, input),
									Type:    "tool_use",
								})
							}
						}
					}
				}
			}
		}
	}

	return conversations
}

// extractToolUses extracts tool usage information
func (lp *LogProcessor) extractToolUses(entries []JSONLogEntry) []ToolUse {
	var toolUses []ToolUse
	toolMap := make(map[string]*ToolUse) // Map tool_use_id to ToolUse

	for _, entry := range entries {
		if entry.Type == "assistant" && entry.Message != nil {
			if content, ok := entry.Message["content"].([]interface{}); ok {
				for _, item := range content {
					if contentItem, ok := item.(map[string]interface{}); ok {
						if contentItem["type"] == "tool_use" {
							toolUse := &ToolUse{
								Success: false,
							}

							if name, ok := contentItem["name"].(string); ok {
								toolUse.Name = name
							}

							if input, ok := contentItem["input"].(map[string]interface{}); ok {
								if inputStr, err := json.MarshalIndent(input, "", "  "); err == nil {
									toolUse.Input = string(inputStr)
								}
							}

							if id, ok := contentItem["id"].(string); ok {
								toolMap[id] = toolUse
								toolUses = append(toolUses, *toolUse)
							}
						}
					}
				}
			}
		} else if entry.Type == "user" && entry.Message != nil {
			if content, ok := entry.Message["content"].([]interface{}); ok {
				for _, item := range content {
					if contentItem, ok := item.(map[string]interface{}); ok {
						if contentItem["type"] == "tool_result" {
							if toolUseID, ok := contentItem["tool_use_id"].(string); ok {
								if toolUse, exists := toolMap[toolUseID]; exists {
									if result, ok := contentItem["content"].(string); ok {
										toolUse.Output = result
										if isError, ok := contentItem["is_error"].(bool); ok {
											toolUse.Success = !isError
										} else {
											toolUse.Success = true // Default to success if is_error is not present
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return toolUses
}

// extractResults extracts the final results
func (lp *LogProcessor) extractResults(entries []JSONLogEntry) *Result {
	var totalCost float64
	var totalDuration int64
	var lastResult *Result
	var foundTotalCost bool

	for _, entry := range entries {
		// Process cost information
		totalCost, foundTotalCost = lp.processCostInfo(entry, totalCost, foundTotalCost)

		// Process duration information
		totalDuration = lp.processDurationInfo(entry, totalDuration)

		// Process result entries
		if entry.Type == "result" {
			lastResult = lp.processResultEntry(entry)
		}
	}

	return lp.buildFinalResult(lastResult, totalCost, totalDuration)
}

// extractOperationFlow extracts the chronological flow of operations
func (lp *LogProcessor) extractOperationFlow(entries []JSONLogEntry) []OperationStep {
	var steps []OperationStep
	stepNumber := 1
	toolMap := make(map[string]string) // tool_use_id to tool name mapping

	for _, entry := range entries {
		switch entry.Type {
		case "system":
			if entry.Subtype == "init" {
				steps = append(steps, OperationStep{
					StepNumber: stepNumber,
					Type:       "system",
					Actor:      "system",
					Content:    "Claude session initialized",
					Success:    true,
					Timestamp:  entry.Timestamp,
				})
				stepNumber++
			}

		case "assistant":
			if entry.Message != nil {
				if content, ok := entry.Message["content"].([]interface{}); ok {
					for _, item := range content {
						if contentItem, ok := item.(map[string]interface{}); ok {
							switch contentItem["type"] {
							case "text":
								if text, ok := contentItem["text"].(string); ok {
									steps = append(steps, OperationStep{
										StepNumber: stepNumber,
										Type:       "assistant_message",
										Actor:      "assistant",
										Content:    lp.truncateString(text, maxDisplayLength),
										Details:    text,
										Success:    true,
										Timestamp:  entry.Timestamp,
									})
									stepNumber++
								}

							case "tool_use":
								toolName := "Unknown Tool"
								if name, ok := contentItem["name"].(string); ok {
									toolName = name
								}

								toolInput := ""
								if input, ok := contentItem["input"].(map[string]interface{}); ok {
									if inputStr, err := json.MarshalIndent(input, "", "  "); err == nil {
										toolInput = string(inputStr)
									}
								}

								// Store tool mapping for later reference
								if id, ok := contentItem["id"].(string); ok {
									toolMap[id] = toolName
								}

								steps = append(steps, OperationStep{
									StepNumber: stepNumber,
									Type:       "tool_use",
									Actor:      "assistant",
									Content:    fmt.Sprintf("Using %s", toolName),
									Details:    toolInput,
									Success:    true,
									Timestamp:  entry.Timestamp,
								})
								stepNumber++
							}
						}
					}
				}
			}

		case "user":
			if entry.Message != nil {
				if content, ok := entry.Message["content"].([]interface{}); ok {
					for _, item := range content {
						if contentItem, ok := item.(map[string]interface{}); ok {
							if contentItem["type"] == "tool_result" {
								toolName := "Tool"
								if toolUseID, ok := contentItem["tool_use_id"].(string); ok {
									if name, exists := toolMap[toolUseID]; exists {
										toolName = name
									}
								}

								isError := false
								if errorFlag, ok := contentItem["is_error"].(bool); ok {
									isError = errorFlag
								}

								resultContent := "No output"
								if result, ok := contentItem["content"].(string); ok {
									resultContent = lp.truncateString(result, maxDisplayLength)
								}

								statusIcon := "✓"
								if isError {
									statusIcon = "✗"
								}

								steps = append(steps, OperationStep{
									StepNumber: stepNumber,
									Type:       "tool_result",
									Actor:      "user",
									Content:    fmt.Sprintf("%s %s result", statusIcon, toolName),
									Details:    resultContent,
									Success:    !isError,
									Timestamp:  entry.Timestamp,
								})
								stepNumber++
							}
						}
					}
				}
			}

		case "result":
			statusIcon := "✓"
			resultType := "Completed"
			if _, ok := entry.Raw["error"].(string); ok {
				statusIcon = "✗"
				resultType = "Failed"
			}

			steps = append(steps, OperationStep{
				StepNumber: stepNumber,
				Type:       "result",
				Actor:      "system",
				Content:    fmt.Sprintf("%s Execution %s", statusIcon, resultType),
				Details:    entry.Result,
				Success:    statusIcon == "✓",
				Timestamp:  entry.Timestamp,
			})
			stepNumber++
		}
	}

	return steps
}

// formatExecution formats the execution into human-readable output
func (lp *LogProcessor) formatExecution(metadata *ExecutionMetadata, conversations []Conversation, toolUses []ToolUse, results *Result, operationFlow []OperationStep) string {
	var output strings.Builder

	// 1. Prompt - simplified to just show the content without header
	actualPrompt := lp.extractActualPrompt(metadata.Prompt)
	output.WriteString(fmt.Sprintf("💬 Prompt:\n%s", actualPrompt))

	// 2. Claude's Response
	if len(conversations) > 0 {
		output.WriteString("\n\n🤖 Claude's Response:\n")
		for _, conv := range conversations {
			if conv.Type == "text" {
				output.WriteString(conv.Content)
			}
		}
		output.WriteString("\n")
	}

	// 3. Operation Flow - enhanced with more detailed information
	if len(operationFlow) > 0 {
		output.WriteString("\n\n⚡ Operation Flow:\n")

		// Group operations by type for better visualization
		systemSteps := 0
		assistantSteps := 0
		toolSteps := 0

		for _, step := range operationFlow {
			icon := lp.getStepIcon(step.Type)
			timestamp := ""
			if step.Timestamp != "" {
				// Parse and format timestamp for better readability
				if t, err := lp.parseTimestamp(step.Timestamp); err == nil {
					timestamp = fmt.Sprintf(" [%s]", t.Format("15:04:05"))
				}
			}

			// Enhanced step display with more context
			output.WriteString(fmt.Sprintf("%d. %s %s%s", step.StepNumber, icon, step.Content, timestamp))

			// Add success indicator for non-system steps
			if step.Type != "system" {
				if step.Success {
					output.WriteString(" ✅")
				} else {
					output.WriteString(" ❌")
				}
			}
			output.WriteString("\n")

			// Show enhanced details
			if step.Details != "" {
				switch step.Type {
				case "tool_use":
					toolSteps++
					// Show tool input details
					if cmd := lp.extractCommandFromDetails(step.Details); cmd != "" {
						output.WriteString(fmt.Sprintf("   ➤ Command: %s\n", cmd))
					} else {
						// Show formatted input for non-bash tools
						formattedInput := lp.formatToolInput(step.Details)
						if formattedInput != "" {
							output.WriteString(fmt.Sprintf("   ➤ Input: %s\n", formattedInput))
						}
					}
				case "tool_result":
					// Show result summary
					if !step.Success {
						output.WriteString(fmt.Sprintf("   ⚠️  Error: %s\n", lp.truncateString(step.Details, maxDisplayLength)))
					} else {
						// Show successful result summary
						summary := lp.summarizeToolResult(step.Details)
						if summary != "" {
							output.WriteString(fmt.Sprintf("   ✓ Result: %s\n", summary))
						}
					}
				case "assistant_message":
					assistantSteps++
					// Show message type and length
					messageLen := len(step.Details)
					if messageLen > maxDisplayLength {
						output.WriteString(fmt.Sprintf("   📝 Message (%d chars): %s...\n",
							messageLen, lp.truncateString(step.Details, maxDisplayLength)))
					}
				case "system":
					systemSteps++
				}
			}
		}

		// Add operation summary
		output.WriteString(fmt.Sprintf("\n📊 Flow Summary: %d system, %d assistant, %d tools used\n",
			systemSteps, assistantSteps, toolSteps))
	}

	// Total Cost Information - as a separate section
	totalCost := metadata.CostUSD
	if results != nil && results.CostUSD > 0 {
		totalCost = results.CostUSD
	}

	output.WriteString(fmt.Sprintf("\n\n💰 Total Cost:\n$%.4f", totalCost))

	// Final Result/Summary - only show if different from response
	if results != nil && results.Message != "" {
		// Only show summary if it's different from the Claude response
		if len(conversations) == 0 || (len(conversations) > 0 && results.Message != conversations[len(conversations)-1].Content) {
			if results.Success {
				output.WriteString(fmt.Sprintf("\n\n📊 Summary:\n%s", results.Message))
			} else {
				output.WriteString(fmt.Sprintf("\n\n❌ Error:\n%s", results.Message))
			}
		}
	}

	return output.String()
}

// extractActualPrompt extracts the actual user prompt content,
// removing execution metadata and showing only content starting from "# Task:"
func (lp *LogProcessor) extractActualPrompt(fullPrompt string) string {
	lines := strings.Split(fullPrompt, "\n")

	// Find the line starting with "# Task:"
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# Task:") {
			// Return everything from this line onwards
			return strings.Join(lines[i:], "\n")
		}
	}

	// If no "# Task:" found, return the full prompt as fallback
	return fullPrompt
}

// getStepIcon returns an appropriate icon for the operation step type
func (lp *LogProcessor) getStepIcon(stepType string) string {
	switch stepType {
	case "system":
		return "🔧"
	case "assistant_message":
		return "💭"
	case "tool_use":
		return "⚡"
	case "tool_result":
		return "📋"
	case "result":
		return "🎯"
	default:
		return "📌"
	}
}

// extractCommandFromDetails extracts bash command from tool input details
func (lp *LogProcessor) extractCommandFromDetails(details string) string {
	inputData := lp.parseJSONInput(details)
	if inputData == nil {
		return ""
	}

	if cmd, ok := inputData["command"].(string); ok {
		return cmd
	}

	return ""
}

// truncateString truncates a string to a maximum length
func (lp *LogProcessor) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// parseTimestamp parses timestamp strings from log entries
func (lp *LogProcessor) parseTimestamp(timestamp string) (*time.Time, error) {
	// Try different timestamp formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestamp); err == nil {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("unable to parse timestamp: %s", timestamp)
}

// formatToolInput formats tool input for display
func (lp *LogProcessor) formatToolInput(input string) string {
	inputData := lp.parseJSONInput(input)
	if inputData == nil {
		return ""
	}

	// Extract key information based on common tool patterns
	if pattern, ok := inputData["pattern"].(string); ok {
		if path, pathOk := inputData["path"].(string); pathOk {
			return fmt.Sprintf("Search '%s' in %s", pattern, path)
		}
		return fmt.Sprintf("Search '%s'", pattern)
	}

	if filePath, ok := inputData["file_path"].(string); ok {
		if _, newOk := inputData["new_string"].(string); newOk {
			return fmt.Sprintf("Edit %s", filePath)
		}
		return fmt.Sprintf("Read %s", filePath)
	}

	if content, ok := inputData["content"].(string); ok {
		return fmt.Sprintf("Write content (%d chars)", len(content))
	}

	// Fall back to truncated JSON
	if inputStr, err := json.Marshal(inputData); err == nil {
		return lp.truncateString(string(inputStr), maxInputLength)
	}

	return ""
}

// summarizeToolResult provides a summary of tool results
func (lp *LogProcessor) summarizeToolResult(result string) string {
	if result == "" {
		return ""
	}

	lines := strings.Split(result, "\n")

	// For file listings
	if strings.Contains(result, "files") && strings.Contains(result, "directories") {
		return fmt.Sprintf("Listed %d items", len(lines))
	}

	// For search results
	if strings.Contains(result, "matches found") {
		return strings.Split(result, "\n")[0] // First line usually contains summary
	}

	// For short results, return as-is
	if len(result) <= shortResultThreshold {
		return strings.TrimSpace(result)
	}

	// For longer results, provide summary
	if len(lines) > longOutputThreshold {
		return fmt.Sprintf("Output: %d lines (first: %s...)", len(lines), lp.truncateString(lines[0], maxFirstLineLength))
	}

	return lp.truncateString(result, maxSummaryLength)
}

// parseJSONInput is a helper function to parse JSON input strings
func (lp *LogProcessor) parseJSONInput(input string) map[string]interface{} {
	var inputData map[string]interface{}
	if err := json.Unmarshal([]byte(input), &inputData); err != nil {
		return nil
	}
	return inputData
}

// processCostInfo processes cost information from log entries
func (lp *LogProcessor) processCostInfo(entry JSONLogEntry, totalCost float64, foundTotalCost bool) (float64, bool) {
	// Check raw data first for total_cost_usd (which is the final cost)
	if entry.Raw != nil {
		if rawCost, ok := entry.Raw["total_cost_usd"].(float64); ok && rawCost > 0 {
			return rawCost, true // Use total cost directly, don't accumulate
		}
	}

	// If no total cost found yet, accumulate individual costs
	if !foundTotalCost {
		if entry.CostUSD > 0 {
			totalCost += entry.CostUSD
		}

		// Also check raw data for individual cost information
		if entry.Raw != nil {
			if rawCost, ok := entry.Raw["cost_usd"].(float64); ok && rawCost > 0 {
				totalCost += rawCost
			}
		}
	}

	return totalCost, foundTotalCost
}

// processDurationInfo processes duration information from log entries
func (lp *LogProcessor) processDurationInfo(entry JSONLogEntry, totalDuration int64) int64 {
	// Accumulate duration
	if entry.Duration > 0 {
		totalDuration += entry.Duration
	}

	// Also check raw data for duration
	if entry.Raw != nil {
		if rawDuration, ok := entry.Raw["duration_ms"].(float64); ok && rawDuration > 0 {
			totalDuration += int64(rawDuration)
		}
	}

	return totalDuration
}

// processResultEntry processes a result entry and creates a Result object
func (lp *LogProcessor) processResultEntry(entry JSONLogEntry) *Result {
	result := &Result{
		Success:  true,
		CostUSD:  entry.CostUSD,
		Duration: entry.Duration,
	}

	if entry.Result != "" {
		result.Message = entry.Result
	}

	// Check for error in raw data
	if errorStr, ok := entry.Raw["error"].(string); ok {
		result.Success = false
		result.Message = errorStr
	}

	return result
}

// buildFinalResult builds the final result with accumulated costs and duration
func (lp *LogProcessor) buildFinalResult(lastResult *Result, totalCost float64, totalDuration int64) *Result {
	// Return result with accumulated costs, or create one if none found
	if lastResult != nil {
		if totalCost > lastResult.CostUSD {
			lastResult.CostUSD = totalCost
		}
		if totalDuration > lastResult.Duration {
			lastResult.Duration = totalDuration
		}
		return lastResult
	}

	// If no result entry found but we have costs, create a summary result
	if totalCost > 0 || totalDuration > 0 {
		return &Result{
			Success:  true,
			CostUSD:  totalCost,
			Duration: totalDuration,
			Message:  "",
		}
	}

	return nil
}
