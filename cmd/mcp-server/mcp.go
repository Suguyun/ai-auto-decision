package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	configFile = "config.json" // 本地运行时使用相对路径
	mu         sync.Mutex
)

type Config struct {
	CPUAlertThreshold float64 `json:"cpu_alert_threshold"`
}

func loadConfig() (*Config, error) {
	// 获取可执行文件所在目录
	execPath, err := os.Executable()
	if err != nil {
		// 如果无法获取可执行文件路径，则使用当前工作目录
		execPath, _ = os.Getwd()
	}
	
	// 构建配置文件的绝对路径
	dir := filepath.Dir(execPath)
	configPath := filepath.Join(dir, configFile)
	
	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 如果配置文件不存在，创建默认配置文件
		defaultConfig := &Config{CPUAlertThreshold: 80.0}
		if err := saveConfig(defaultConfig); err != nil {
			log.Printf("Failed to create default config file: %v", err)
			return defaultConfig, nil
		}
		log.Printf("Created default config file at: %s", configPath)
		return defaultConfig, nil
	}
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		// 默认配置
		return &Config{CPUAlertThreshold: 80.0}, nil
	}
	var cfg Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		log.Printf("Failed to parse config file, using default config: %v", err)
		return &Config{CPUAlertThreshold: 80.0}, nil
	}
	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	// 获取可执行文件所在目录
	execPath, err := os.Executable()
	if err != nil {
		// 如果无法获取可执行文件路径，则使用当前工作目录
		execPath, _ = os.Getwd()
	}
	
	// 构建配置文件的绝对路径
	dir := filepath.Dir(execPath)
	configPath := filepath.Join(dir, configFile)
	
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

// MCP 工具：adjust_cpu_threshold
func adjustThreshold(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	value, ok := args["value"].(float64)
	if !ok {
		return mcp.NewToolResultError("parameter 'value' must be a number"), nil
	}

	if value < 0 || value > 100 {
		return mcp.NewToolResultError("'value' must be between 0 and 100"), nil
	}

	mu.Lock()
	defer mu.Unlock()

	cfg, err := loadConfig()
	if err != nil {
		return mcp.NewToolResultError("failed to load config"), nil
	}

	old := cfg.CPUAlertThreshold
	cfg.CPUAlertThreshold = value
	if err := saveConfig(cfg); err != nil {
		return mcp.NewToolResultError("failed to save config: " + err.Error()), nil
	}

	log.Printf("✅ [MCP] 阈值已从 %.1f 更新为 %.1f", old, value)

	result := map[string]interface{}{
		"old_value": old,
		"new_value": value,
		"status":    "success",
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(jsonResult)),
		},
	}, nil
}

func main() {
	mcpServer := server.NewMCPServer("auto-config-agent", "1.0.0")

	mcpServer.AddTool(mcp.Tool{
		Name:        "adjust_cpu_threshold",
		Description: "动态调整 CPU 告警阈值（0~100）",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"value": map[string]interface{}{
					"type":        "number",
					"description": "新的阈值百分比",
					"minimum":     0,
					"maximum":     100,
				},
			},
			Required: []string{"value"},
		},
	}, adjustThreshold)

	httpServer := server.NewStreamableHTTPServer(mcpServer)
	log.Println("🚀 MCP Server listening on :9001/mcp")
	log.Fatal(httpServer.Start(":9001"))
}