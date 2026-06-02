// Package mcp implements the Model Context Protocol (MCP) server for RyxoGo.
// Cursor, Claude Code, and other AI tools connect to this via JSON-RPC 2.0.
//
// MCP spec: https://modelcontextprotocol.io/specification
// Transport: stdio (primary) + SSE HTTP (for Cursor remote)
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ---------------------------------------------------------
// JSON-RPC 2.0 types
// ---------------------------------------------------------

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type Notification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ---------------------------------------------------------
// MCP protocol types
// ---------------------------------------------------------

type InitializeParams struct {
	ProtocolVersion string     `json:"protocolVersion"`
	Capabilities    ClientCaps `json:"capabilities"`
	ClientInfo      ClientInfo `json:"clientInfo"`
}

type ClientCaps struct{}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	ProtocolVersion string     `json:"protocolVersion"`
	Capabilities    ServerCaps `json:"capabilities"`
	ServerInfo      ServerInfo `json:"serverInfo"`
}

type ServerCaps struct {
	Tools     *ToolsCap     `json:"tools,omitempty"`
	Resources *ResourcesCap `json:"resources,omitempty"`
}

type ToolsCap     struct{ ListChanged bool `json:"listChanged"` }
type ResourcesCap struct{ Subscribe bool `json:"subscribe"`; ListChanged bool `json:"listChanged"` }

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ToolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type CallToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ---------------------------------------------------------
// StdioServer — runs MCP over stdin/stdout (what Cursor uses)
// ---------------------------------------------------------

type StdioServer struct {
	handler *Server
	writer  *json.Encoder
}

func NewStdioServer(projectRoot string) *StdioServer {
	return &StdioServer{
		handler: NewServer(projectRoot),
		writer:  json.NewEncoder(os.Stdout),
	}
}

// Run reads JSON-RPC messages from stdin and writes responses to stdout.
// This is the stdio transport required by Cursor's MCP client.
func (s *StdioServer) Run() error {
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(nil, -32700, "Parse error")
			continue
		}

		s.handle(&req)
	}
}

func (s *StdioServer) handle(req *Request) {
	switch req.Method {

	case "initialize":
		var params InitializeParams
		json.Unmarshal(req.Params, &params)
		s.send(req.ID, InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: ServerCaps{
				Tools: &ToolsCap{ListChanged: false},
			},
			ServerInfo: ServerInfo{
				Name:    "ryxogo-mcp",
				Version: "0.1.7",
			},
		})

	case "notifications/initialized":
		// Client confirmed init — no response needed

	case "tools/list":
		s.send(req.ID, map[string]interface{}{
			"tools": s.handler.Tools(),
		})

	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendError(req.ID, -32602, "Invalid params")
			return
		}

		var args map[string]interface{}
		json.Unmarshal(params.Arguments, &args)

		result, err := s.handler.Call(params.Name, args)
		if err != nil {
			s.send(req.ID, CallToolResult{
				Content: []ContentBlock{{Type: "text", Text: "Error: " + err.Error()}},
				IsError: true,
			})
			return
		}

		// Serialize result to JSON string for text content
		text, _ := json.MarshalIndent(result, "", "  ")
		s.send(req.ID, CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: string(text)}},
		})

	case "ping":
		s.send(req.ID, map[string]string{})

	default:
		s.sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (s *StdioServer) send(id interface{}, result interface{}) {
	s.writer.Encode(Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func (s *StdioServer) sendError(id interface{}, code int, msg string) {
	s.writer.Encode(Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: msg},
	})
}
