package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type toolDef struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
	Execute     func(argsJSON string) string
}

var tools []toolDef

func init() {
	tools = append(tools, toolDef{
		Name:        "write_file",
		Description: "Write content to a file at the specified path. Creates parent directories if they don't exist.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Full file path to write to (e.g. C:\\projects\\tae\\essay.txt)",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content to write to the file",
				},
			},
			"required": []interface{}{"path", "content"},
		},
		Execute: executeWriteFile,
	})
}

func executeWriteFile(argsJSON string) string {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("Failed to parse arguments: %v", err)
	}
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	if path == "" {
		return "Missing 'path' argument"
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Sprintf("Failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Sprintf("Failed to write file %s: %v", path, err)
	}
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path)
}

func executeToolCall(name string, argsJSON string) string {
	for _, t := range tools {
		if t.Name == name {
			return t.Execute(argsJSON)
		}
	}
	return fmt.Sprintf("Unknown tool: %s", name)
}

func formatToolsForProvider(provider string) interface{} {
	switch provider {
	case "anthropic":
		var result []map[string]interface{}
		for _, t := range tools {
			result = append(result, map[string]interface{}{
				"name":        t.Name,
				"description": t.Description,
				"input_schema": map[string]interface{}{
					"type":       "object",
					"properties": t.Parameters["properties"],
					"required":   t.Parameters["required"],
				},
			})
		}
		return result
	case "gemini":
		var funcs []map[string]interface{}
		for _, t := range tools {
			funcs = append(funcs, map[string]interface{}{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			})
		}
		return []map[string]interface{}{
			{"functionDeclarations": funcs},
		}
	default:
		var result []map[string]interface{}
		for _, t := range tools {
			result = append(result, map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  t.Parameters,
				},
			})
		}
		return result
	}
}

func parseToolCalls(provider string, body []byte) []toolCall {
	switch provider {
	case "anthropic":
		return parseAnthropicToolCalls(body)
	case "gemini":
		return parseGeminiToolCalls(body)
	default:
		return parseOpenAIToolCalls(body)
	}
}

func parseOpenAIToolCalls(body []byte) []toolCall {
	var resp struct {
		Choices []struct {
			Message struct {
				ToolCalls []toolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}
	if len(resp.Choices) == 0 {
		return nil
	}
	return resp.Choices[0].Message.ToolCalls
}

func parseAnthropicToolCalls(body []byte) []toolCall {
	var resp struct {
		Content []json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}
	var calls []toolCall
	for _, raw := range resp.Content {
		var block struct {
			Type  string          `json:"type"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		}
		if err := json.Unmarshal(raw, &block); err != nil {
			continue
		}
		if block.Type == "tool_use" {
			args, _ := json.Marshal(block.Input)
			calls = append(calls, toolCall{
				ID:   block.ID,
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      block.Name,
					Arguments: string(args),
				},
			})
		}
	}
	return calls
}

func parseGeminiToolCalls(body []byte) []toolCall {
	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []json.RawMessage `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}
	if len(resp.Candidates) == 0 {
		return nil
	}
	var calls []toolCall
	for _, raw := range resp.Candidates[0].Content.Parts {
		var block struct {
			FuncCall *struct {
				Name string          `json:"name"`
				Args json.RawMessage `json:"args"`
			} `json:"functionCall"`
		}
		if err := json.Unmarshal(raw, &block); err != nil {
			continue
		}
		if block.FuncCall != nil {
			args, _ := json.Marshal(block.FuncCall.Args)
			calls = append(calls, toolCall{
				ID:   block.FuncCall.Name,
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      block.FuncCall.Name,
					Arguments: string(args),
				},
			})
		}
	}
	return calls
}

func extractFinalText(provider string, body []byte) string {
	switch provider {
	case "anthropic":
		var resp struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return ""
		}
		var texts []string
		for _, c := range resp.Content {
			if c.Type == "text" {
				texts = append(texts, c.Text)
			}
		}
		return strings.TrimSpace(strings.Join(texts, "\n"))
	case "gemini":
		var resp struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return ""
		}
		if len(resp.Candidates) > 0 {
			var texts []string
			for _, p := range resp.Candidates[0].Content.Parts {
				if p.Text != "" {
					texts = append(texts, p.Text)
				}
			}
			return strings.TrimSpace(strings.Join(texts, "\n"))
		}
		return ""
	default:
		var resp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return ""
		}
		if len(resp.Choices) > 0 {
			return resp.Choices[0].Message.Content
		}
		return ""
	}
}
