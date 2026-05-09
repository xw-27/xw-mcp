package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"xw-mcp/internal/config"
	"xw-mcp/internal/plugin"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// McpServer MCP 服务结构体
type McpServer struct {
	server *mcp.Server
	pm     *plugin.PluginManager
	cfg    *config.ConfigManager
	port   int
}

func New(pm *plugin.PluginManager, cfg *config.ConfigManager) *McpServer {
	port := 8080
	if cfg != nil {
		if portVal, ok := cfg.Get("server.port"); ok {
			port = portVal.Int()
		}
	}

	s := &McpServer{
		pm:   pm,
		cfg:  cfg,
		port: port,
	}

	// 创建 MCP Server
	s.server = mcp.NewServer(&mcp.Implementation{
		Name:    ServerName,
		Version: Version,
	}, nil)

	// 注册初始 tools
	for _, p := range pm.Plugins() {
		s.addTool(p)
	}

	// 事件驱动：插件变化时更新 MCP tools
	pm.AddEventHandler(func(data plugin.PluginEventData) {
		switch data.Event {
		case plugin.EventPluginAdd:
			s.addTool(data.Plugin)
		case plugin.EventPluginDelete:
			s.server.RemoveTools(data.OldPlugin.Name)
		case plugin.EventPluginUpdate:
			s.server.RemoveTools(data.OldPlugin.Name)
			s.addTool(data.Plugin)
		}
	})

	return s
}

// addTool 注册单个工具到 MCP Server
func (s *McpServer) addTool(p *plugin.Plugin) {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        p.Name,
		Description: p.Description,
		InputSchema: p.InputSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		result, err := p.Execute(input)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("error: %v", err)},
				},
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("%v", result)},
			},
		}, nil, nil
	})
}

// Run 启动 MCP HTTP 服务
func (s *McpServer) Run() error {
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s.server
	}, nil)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: handler,
	}

	log.Printf("[mcp] server starting on port %d...", s.port)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("mcp server run failed: %w", err)
	}
	return nil
}

// Close 关闭 MCP 服务
func (s *McpServer) Close() error {
	if s.pm != nil {
		s.pm.Close()
	}
	return nil
}