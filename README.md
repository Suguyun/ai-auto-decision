# AI 自动决策系统

本系统是一个基于大语言模型的自动决策系统，能够监控系统指标并在必要时自动调整配置。

## 系统架构

系统由以下组件构成：

1. **Agent** - 决策代理，负责监控系统指标并调用 LLM 进行决策
2. **MCP Server** - 配置管理工具服务端，提供可执行的工具函数
3. **LLM Service** - 大语言模型服务（外部依赖）

## 工作流程

1. Agent 定期采集系统指标（如 CPU 使用率）
2. 将指标数据发送给 LLM 进行分析
3. LLM 根据预设规则判断是否需要调整配置
4. 如需调整，LLM 会调用 MCP Server 提供的工具函数
5. MCP Server 执行具体操作并更新配置

## 部署说明

### 方式一：使用 Docker Compose 部署（推荐）

```bash
docker-compose up --build
```

在运行前，请设置以下环境变量：
- `LLM_API_KEY`: 阿里百炼平台的API Key

可以通过以下方式设置环境变量：

1. 复制 `.env.example` 文件为 `.env`：
   ```bash
   cp .env.example .env
   ```

2. 编辑 `.env` 文件，填入你的API Key：
   ```
   LLM_API_KEY=你的_api_key
   ```

3. 启动服务：
   ```bash
   docker-compose up --build
   ```

### 方式二：直接运行

1. 编译 Agent 和 MCP Server：
   ```bash
   go build -o agent ./cmd/agent
   go build -o mcp-server ./cmd/mcp-server
   ```

2. 设置环境变量并运行服务：
   ```bash
   export LLM_API_KEY=你的_api_key
   ./mcp-server &
   ./agent
   ```

## 配置文件

系统使用 `config.json` 存储配置信息，默认内容如下：

```json
{
  "cpu_alert_threshold": 85.0
}
```

## 依赖项

- Go 1.24.5
- github.com/mark3labs/mcp-go v0.43.2
- github.com/sashabaranov/go-openai v1.41.2
=======
# ai-auto-decision
本系统是一个基于大语言模型的自动决策系统，能够监控系统指标并在必要时自动调整配置。