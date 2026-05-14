package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"xw-mcp/internal/bundle"
	"xw-mcp/internal/config"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// McpServer MCP 服务结构体
// 持有 bundle.BundleContext 引用，通过 AfterLoad 回调实现与 bundle 系统的联动
type McpServer struct {
	server    *mcp.Server        // 底层 MCP Server
	bundleCtx *bundle.BundleContext  // Bundle 上下文
	cfg       *config.ConfigWatcher  // 配置监听器
	port      int                 // HTTP 服务端口
}

// New 创建 MCP Server 实例
// bundleCtx: Bundle 上下文，用于注册 AfterLoad 回调
// cfg: 配置管理器，可选
func New(bundleCtx *bundle.BundleContext, cfg *config.ConfigWatcher) *McpServer {
	port := 8080
	if cfg != nil {
		if portVal, ok := cfg.Get("server.port"); ok {
			port = portVal.ToInt()
		}
	}

	s := &McpServer{
		bundleCtx: bundleCtx,
		cfg:       cfg,
		port:      port,
	}

	// 创建 MCP Server
	s.server = mcp.NewServer(&mcp.Implementation{
		Name:    ServerName,
		Version: Version,
	}, nil)

	// 注册 Bundle 加载完成回调
	// 当任意 bundle 完成 MetaLoad 时，reloadAll 会被调用
	bundleCtx.AfterLoad(func(ctx *bundle.BundleContext) {
		s.reloadAll(ctx)
	})

	return s
}

// reloadAll 全量重载所有 actions 到 MCP Server
// 1. 先移除所有已注册的 tools/prompts/resources/templates
// 2. 再从当前所有 bundle 重新注册
func (s *McpServer) reloadAll(ctx *bundle.BundleContext) {
	log.Printf("[mcp] reloading all actions...")

	// 1. 移除所有已注册的 actions
	for _, b := range ctx.List() {
		for _, t := range b.Registry().ListTools() {
			name := b.Name() + "_" + t.Name
			s.server.RemoveTools(name)
		}
		for _, p := range b.Registry().ListPrompts() {
			name := b.Name() + "_" + p.Name
			s.server.RemovePrompts(name)
		}
		for _, r := range b.Registry().ListResources() {
			name := b.Name() + "_" + r.Name
			s.server.RemoveResources(name)
		}
		for _, t := range b.Registry().ListTemplates() {
			name := b.Name() + "_" + t.Name
			s.server.RemoveResourceTemplates(name)
		}
	}

	// 2. 重新注册所有 actions
	for _, b := range ctx.List() {
		for _, t := range b.Registry().ListTools() {
			s.addTool(b.Name(), t)
		}
		for _, p := range b.Registry().ListPrompts() {
			s.addPrompt(b.Name(), p)
		}
		for _, r := range b.Registry().ListResources() {
			s.addResource(b.Name(), r)
		}
		for _, t := range b.Registry().ListTemplates() {
			s.addTemplate(b.Name(), t)
		}
	}

	log.Printf("[mcp] reload complete")
}

// addTool 注册工具到 MCP Server
// bundleName: bundle 名称
// t: 工具 action
// 工具名称格式: bundleName_actionName
func (s *McpServer) addTool(bundleName string, t *bundle.ToolAction) {
	name := bundleName + "_" + t.Name

	s.server.AddTool(&mcp.Tool{
		Name:        name,
		Description: t.Description,
		InputSchema: t.InputSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Printf("[mcp] tool called: name=%s, arguments=%v", name, req.Params.Arguments)

		// 解析 json.RawMessage 为 map[string]interface{}
		var params map[string]interface{}
		if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
			log.Printf("[mcp] tool arguments parse failed: name=%s, error=%v", name, err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("parse arguments failed: %v", err)}},
			}, nil
		}

		result, err := s.bundleCtx.DoAction(bundle.ActionTypeTool, name, params)
		if err != nil {
			log.Printf("[mcp] tool error: name=%s, error=%v", name, err)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("error: %v", err)}},
			}, nil
		}
		log.Printf("[mcp] tool result: name=%s, result=%v", name, result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("%v", result)}},
		}, nil
	})
}

// addPrompt 注册提示词到 MCP Server
// bundleName: bundle 名称
// p: 提示词 action
// 提示词名称格式: bundleName_actionName
func (s *McpServer) addPrompt(bundleName string, p *bundle.PromptAction) {
	name := bundleName + "_" + p.Name

	s.server.AddPrompt(&mcp.Prompt{
		Name:        name,
		Description: p.Description,
		Arguments:   toMcpArguments(p.Arguments),
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Prompt 的 Arguments 已经是 map[string]string
		// 转换为 map[string]interface{} 以保持一致性
		params := make(map[string]interface{})
		for k, v := range req.Params.Arguments {
			params[k] = v
		}

		result, err := s.bundleCtx.DoAction(bundle.ActionTypePrompt, name, params)
		if err != nil {
			log.Printf("[mcp] prompt error: name=%s, error=%v", name, err)
			return nil, err
		}
		if msgs, ok := result.([]*bundle.PromptMessage); ok {
			return toMcpPromptResult(msgs), nil
		}
		return nil, fmt.Errorf("invalid prompt result")
	})
}

// addResource 注册资源到 MCP Server
// bundleName: bundle 名称
// r: 资源 action
// 资源名称格式: bundleName_actionName
func (s *McpServer) addResource(bundleName string, r *bundle.ResourceAction) {
	name := bundleName + "_" + r.Name

	s.server.AddResource(&mcp.Resource{
		Name:        name,
		Description: r.Description,
		URI:         r.URI,
		MIMEType:    r.MIMEType,
	}, nil)
}

// addTemplate 注册资源模板到 MCP Server
// bundleName: bundle 名称
// t: 资源模板 action
// 模板名称格式: bundleName_actionName
func (s *McpServer) addTemplate(bundleName string, t *bundle.ResourceTemplateAction) {
	name := bundleName + "_" + t.Name

	s.server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        name,
		Description: t.Description,
		URITemplate: t.URITemplate,
		MIMEType:    t.MIMEType,
	}, nil)
}

// toMcpArguments 将 bundle.PromptArgument 转换为 mcp.PromptArgument
func toMcpArguments(args []*bundle.PromptArgument) []*mcp.PromptArgument {
	var result []*mcp.PromptArgument
	for _, a := range args {
		result = append(result, &mcp.PromptArgument{
			Name:        a.Name,
			Description: a.Description,
			Required:    a.Required,
		})
	}
	return result
}

// toMcpPromptResult 将 bundle.PromptMessage 转换为 mcp.GetPromptResult
func toMcpPromptResult(msgs []*bundle.PromptMessage) *mcp.GetPromptResult {
	var result mcp.GetPromptResult
	for _, m := range msgs {
		result.Messages = append(result.Messages, &mcp.PromptMessage{
			Role:    mcp.Role(m.Role),
			Content: &mcp.TextContent{Text: m.Content},
		})
	}
	return &result
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
	return nil
}