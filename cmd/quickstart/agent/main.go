package main

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/components/model"
	callbackHelpers "github.com/cloudwego/eino/utils/callbacks"
	"github.com/coze-dev/cozeloop-go"
	"github.com/joho/godotenv"
	"likeeino/internal/logs"
	"log"
	"os"

	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func main() {

	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}

	//初始化数据,在env配置文件配置即可
	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	openAIModelName := os.Getenv("OPENAI_MODEL_NAME")
	openAIBaseURL := os.Getenv("OPENAI_BASE_URL")

	cozeloopApiToken := os.Getenv("COZELOOP_API_TOKEN")
	cozeloopWorkspaceID := os.Getenv("COZELOOP_WORKSPACE_ID") // use cozeloop trace, from https://loop.coze.cn/open/docs/cozeloop/go-sdk#4a8c980e

	ctx := context.Background()
	var handlers []callbacks.Handler
	if cozeloopApiToken != "" && cozeloopWorkspaceID != "" {
		client, err := cozeloop.NewClient(
			cozeloop.WithAPIToken(cozeloopApiToken),
			cozeloop.WithWorkspaceID(cozeloopWorkspaceID),
		)
		if err != nil {
			panic(err)
		}
		defer client.Close(ctx)
		handlers = append(handlers, clc.NewLoopHandler(client))
	}
	callbacks.AppendGlobalHandlers(handlers...)

	//大模型回调函数
	modelHandler := &callbackHelpers.ModelCallbackHandler{
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *model.CallbackOutput) context.Context {
			// 1. output.Result 类型是 string
			fmt.Println("模型思考过程-------------------------")
			fmt.Println("output.Message.Content:" + output.Message.Content)
			fmt.Println("模型思考过程-------------------------")
			return ctx
		},
	}
	//工具回调函数
	toolHandler := &callbackHelpers.ToolCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *tool.CallbackInput) context.Context {
			fmt.Printf("开始执行工具，工具名:%s,参数: %s\n", info.Name, input.ArgumentsInJSON)
			return ctx
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *tool.CallbackOutput) context.Context {
			fmt.Printf("工具执行完成，工具名:%s,结果: %s\n", info.Name, output.Response)
			return ctx
		},
	}
	//构建实际回调函数Handler
	handler := callbackHelpers.NewHandlerHelper().
		ChatModel(modelHandler).
		Tool(toolHandler).
		Handler()

	updateTool, err := utils.InferTool("update_todo", "Update a todo item, eg: content,deadline...", UpdateTodoFunc)
	if err != nil {
		logs.Errorf("InferTool failed, err=%v", err)
		return
	}

	// 创建 DuckDuckGo 工具
	searchTool, err := duckduckgo.NewTextSearchTool(ctx, &duckduckgo.Config{})
	if err != nil {
		logs.Errorf("NewTextSearchTool failed, err=%v", err)
		return
	}

	// 初始化 tools,三种方式
	todoTools := []tool.BaseTool{
		getAddTodoTool(), // 使用 NewTool 方式
		updateTool,       // 使用 InferTool 方式
		&ListTodoTool{},  // 使用结构体实现方式, 此处未实现底层逻辑
		searchTool,
	}

	// 创建并配置 ChatModel
	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		BaseURL:     openAIBaseURL,
		Model:       openAIModelName,
		APIKey:      openAIAPIKey,
		Temperature: float32(0.7),
	})
	if err != nil {
		logs.Errorf("NewChatModel failed, err=%v", err)
		return
	}

	// 获取工具信息, 用于绑定到 ChatModel
	toolInfos := make([]*schema.ToolInfo, 0, len(todoTools))
	var info *schema.ToolInfo
	for _, todoTool := range todoTools {
		info, err = todoTool.Info(ctx)
		if err != nil {
			logs.Infof("get ToolInfo failed, err=%v", err)
			return
		}
		toolInfos = append(toolInfos, info)
	}

	// 将 tools 绑定到 ChatModel
	err = chatModel.BindTools(toolInfos)
	if err != nil {
		logs.Errorf("BindTools failed, err=%v", err)
		return
	}

	//工具使用,运行大模型调用外部工具完成特定任务
	// 创建 tools 节点
	todoToolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: todoTools,
	})
	if err != nil {
		logs.Errorf("NewToolNode failed, err=%v", err)
		return
	}

	//在编排中使用
	// 构建完整的处理链
	chain := compose.NewChain[[]*schema.Message, []*schema.Message]()
	chain.
		AppendChatModel(chatModel, compose.WithNodeName("chat_model")).
		AppendToolsNode(todoToolsNode, compose.WithNodeName("tools"))

	// 编译并运行 chain
	agent, err := chain.Compile(ctx)
	if err != nil {
		logs.Errorf("chain.Compile failed, err=%v", err)
		return
	}

	// 运行示例
	resp, err := agent.Invoke(ctx, []*schema.Message{
		{
			Role:    schema.User,
			Content: "添加一个学习 Eino 的 TODO，同时搜索一下 cloudwego/eino 的仓库地址",
		},
	}, compose.WithCallbacks(handler))
	if err != nil {
		logs.Errorf("agent.Invoke failed, err=%v", err)
		return
	}

	// 输出结果

	for idx, msg := range resp {
		logs.Infof("\n")
		logs.Infof("message %d: %s: %s", idx, msg.Role, msg.Content)
	}
}

// 获取添加 todo 工具
// 使用 utils.NewTool 创建工具
func getAddTodoTool() tool.InvokableTool {
	info := &schema.ToolInfo{
		Name: "add_todo",
		Desc: "Add a todo item",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"content": {
				Desc:     "The content of the todo item",
				Type:     schema.String,
				Required: true,
			},
			"started_at": {
				Desc: "The started time of the todo item, in unix timestamp",
				Type: schema.Integer,
			},
			"deadline": {
				Desc: "The deadline of the todo item, in unix timestamp",
				Type: schema.Integer,
			},
		}),
	}

	return utils.NewTool(info, AddTodoFunc)
}

// ListTodoTool
// 获取列出 todo 工具
// 自行实现 InvokableTool 接口
type ListTodoTool struct{}

func (lt *ListTodoTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "list_todo",
		Desc: "List all todo items",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"finished": {
				Desc:     "filter todo items if finished",
				Type:     schema.Boolean,
				Required: false,
			},
		}),
	}, nil
}

type TodoUpdateParams struct {
	ID        string  `json:"id" jsonschema_description:"id of the todo"`
	Content   *string `json:"content,omitempty" jsonschema_description:"content of the todo"`
	StartedAt *int64  `json:"started_at,omitempty" jsonschema_description:"start time in unix timestamp"`
	Deadline  *int64  `json:"deadline,omitempty" jsonschema_description:"deadline of the todo in unix timestamp"`
	Done      *bool   `json:"done,omitempty" jsonschema_description:"done status"`
}

type TodoAddParams struct {
	Content  string `json:"content"`
	StartAt  *int64 `json:"started_at,omitempty"` // 开始时间
	Deadline *int64 `json:"deadline,omitempty"`
}

func (lt *ListTodoTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	logs.Infof("invoke tool list_todo: %s", argumentsInJSON)

	// Tool处理代码
	// ...

	return `{"todos": [{"id": "1", "content": "在2024年12月10日之前完成Eino项目演示文稿的准备工作", "started_at": 1717401600, "deadline": 1717488000, "done": false}]}`, nil
}

func AddTodoFunc(_ context.Context, params *TodoAddParams) (string, error) {
	logs.Infof("invoke tool add_todo: %+v", params)

	// Tool处理代码
	// ...

	return `{"msg": "add todo success"}`, nil
}

func UpdateTodoFunc(_ context.Context, params *TodoUpdateParams) (string, error) {
	logs.Infof("invoke tool update_todo: %+v", params)

	// Tool处理代码
	// ...

	return `{"msg": "update todo success"}`, nil
}
