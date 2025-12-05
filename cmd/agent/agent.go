package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/sashabaranov/go-openai"
)

const (
	LLM_ENDPOINT = "https://dashscope.aliyuncs.com/compatible-mode/v1" // 阿里百炼服务地址
	MCP_ENDPOINT = "http://localhost:9001/mcp"
	MODEL_NAME   = "qwen-turbo" // 阿里百炼提供的模型名称
)

// 模拟采集系统指标
func getCurrentMetrics() map[string]float64 {
	return map[string]float64{
		"cpu_usage_percent": 80 + rand.Float64()*15, // 80~95%
	}

}

func main() {
	rand.Seed(time.Now().UnixNano())
	log.Println("🕒 Auto Decision Agent started. Running every 5 seconds...")

	// 初始化 LLM 客户端（兼容 OpenAI API）
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置 LLM_API_KEY 环境变量")
	}
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = LLM_ENDPOINT
	llmClient := openai.NewClientWithConfig(config)

	// 初始化 MCP 客户端
	mcpCli, err := client.NewStreamableHttpClient(MCP_ENDPOINT)
	if err != nil {
		log.Fatalf("Failed to create MCP client: %v", err)
	}

	// 启动 MCP 客户端
	if err := mcpCli.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start MCP client: %v", err)
	}

	// 等待连接建立
	time.Sleep(1 * time.Second)

	// 检查客户端是否已初始化
	if !mcpCli.IsInitialized() {
		// 初始化 MCP 客户端连接
		initReq := mcp.InitializeRequest{
			Params: mcp.InitializeParams{
				ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
				ClientInfo: mcp.Implementation{
					Name:    "auto-decision-agent",
					Version: "1.0.0",
				},
			},
		}

		_, err = mcpCli.Initialize(context.Background(), initReq)
		if err != nil {
			log.Fatalf("Failed to initialize MCP client: %v", err)
		}
	}

	defer mcpCli.Close()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		metrics := getCurrentMetrics()
		cpu := metrics["cpu_usage_percent"]

		prompt := fmt.Sprintf(`
当前系统状态：
- CPU 使用率: %.1f%%

规则：如果 CPU 使用率持续高于 85%%，建议将告警阈值适当调高（例如 90），以避免频繁告警。
请判断是否需要调整配置。如需调整，请调用 adjust_cpu_threshold 工具，并传入新的阈值（数字，0~100）。
`, cpu)

		log.Printf("📊 当前 CPU: %.1f%%", cpu)

		// 调用 LLM
		resp, err := llmClient.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model: MODEL_NAME,
				Messages: []openai.ChatCompletionMessage{
					{Role: openai.ChatMessageRoleUser, Content: prompt},
				},
				Tools: []openai.Tool{
					{
						Type: openai.ToolTypeFunction,
						Function: &openai.FunctionDefinition{
							Name:        "adjust_cpu_threshold",
							Description: "调整 CPU 告警阈值",
							Parameters: map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"value": map[string]string{"type": "number"},
								},
								"required": []string{"value"},
							},
						},
					},
				},
				ToolChoice: "auto",
			},
		)

		if err != nil {
			log.Printf("❌ LLM 调用失败: %v", err)
			continue
		}

		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				if tc.Function.Name == "adjust_cpu_threshold" {
					var args map[string]interface{}
					if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
						log.Printf("⚠️ 参数解析失败: %v", err)
						continue
					}

					log.Printf("🧠 LLM 决策: 调用 %s(%v)", tc.Function.Name, args)

					// 调用 MCP 工具
					params := mcp.CallToolParams{
						Name:      tc.Function.Name,
						Arguments: args,
					}

					request := mcp.CallToolRequest{
						Params: params,
					}

					result, err := mcpCli.CallTool(context.Background(), request)
					if err != nil {
						log.Printf("❌ MCP 执行失败: %v", err)
					} else {
						log.Printf("✅ 配置已更新: %+v", result)
					}
				}
			}
		} else {
			log.Println("🤔 LLM 认为无需调整配置")
		}
	}
}
