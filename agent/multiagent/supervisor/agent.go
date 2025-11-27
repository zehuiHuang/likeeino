package supervisor

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/joho/godotenv"
	"likeeino/pkg/model"
	"likeeino/pkg/trace"
	"log"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/flow/agent/multiagent/plan_execute/tools"
)

func buildSearchAgent(ctx context.Context) (adk.Agent, error) {
	//1、创建chat model
	m := model.NewChatModel()

	type searchReq struct {
		Query string `json:"query"`
	}

	type searchResp struct {
		Result string `json:"result"`
	}

	search := func(ctx context.Context, req *searchReq) (*searchResp, error) {
		return &searchResp{
			Result: "2024年，美国国内生产总值为29.18万亿美元，纽约州国内生产总值为2.297万亿美元",
		}, nil
	}

	searchTool, err := tools.SafeInferTool("search", "在互联网上搜索信息", search)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "research_agent",
		Description: "负责搜索互联网信息的代理",
		Instruction: `
		你是一个搜索代理.

        说明：
          -仅协助完成与研究相关的任务，不要做任何数学题
          -完成任务后，直接回复主管
          -只回复你的工作结果，不要包含任何其他文本。.`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				//设置搜索工具:searchTool
				Tools: []tool.BaseTool{searchTool},
				UnknownToolsHandler: func(ctx context.Context, name, input string) (string, error) {
					return fmt.Sprintf("unknown tool: %s", name), nil
				},
			},
		},
	})
}

// 创建一个agent,他绑定了加乘除三种工具
func buildMathAgent(ctx context.Context) (adk.Agent, error) {
	m := model.NewChatModel()

	type addReq struct {
		A float64 `json:"a"`
		B float64 `json:"b"`
	}

	type addResp struct {
		Result float64
	}

	add := func(ctx context.Context, req *addReq) (*addResp, error) {
		return &addResp{
			Result: req.A + req.B,
		}, nil
	}

	addTool, err := tools.SafeInferTool("add", "add two numbers", add)
	if err != nil {
		return nil, err
	}

	type multiplyReq struct {
		A float64 `json:"a"`
		B float64 `json:"b"`
	}

	type multiplyResp struct {
		Result float64
	}

	multiply := func(ctx context.Context, req *multiplyReq) (*multiplyResp, error) {
		return &multiplyResp{
			Result: req.A * req.B,
		}, nil
	}

	multiplyTool, err := tools.SafeInferTool("multiply", "multiply two numbers", multiply)
	if err != nil {
		return nil, err
	}

	type divideReq struct {
		A float64 `json:"a"`
		B float64 `json:"b"`
	}

	type divideResp struct {
		Result float64
	}

	divide := func(ctx context.Context, req *divideReq) (*divideResp, error) {
		return &divideResp{
			Result: req.A / req.B,
		}, nil
	}

	divideTool, err := tools.SafeInferTool("divide", "divide two numbers", divide)
	if err != nil {
		return nil, err
	}
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "math_agent",
		Description: "当前代理工具负责数学计算",
		Instruction: `
		你是一个数学的代理工具.

        说明：
           -仅协助数学相关任务
           -完成任务后，直接回复主管
           -仅回复您的工作结果，不包括任何其他文本.`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				//将加乘除工具配置到agent上,使其有计算的能力
				Tools: []tool.BaseTool{addTool, multiplyTool, divideTool},
				UnknownToolsHandler: func(ctx context.Context, name, input string) (string, error) {
					return fmt.Sprintf("unknown tool: %s", name), nil
				},
			},
		},
	})
}

// 创建一个主agent,用来管理和协调其他两个子agent
func buildSupervisor(ctx context.Context) (adk.Agent, error) {
	m := model.NewChatModel()

	sv, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "supervisor",
		Description: "负责监督任务的代理人",
		Instruction: `
		您是管理两个代理的主管：

         - 研究代理人。将研究相关任务分配给此代理
         - 数学代理人。将数学相关任务分配给此代理
           一次将工作分配给一个代理，不要并行呼叫代理。
           不要自己做任何工作。.`,
		Model: m,
		Exit:  &adk.ExitTool{},
	})
	if err != nil {
		return nil, err
	}

	searchAgent, err := buildSearchAgent(ctx)
	if err != nil {
		return nil, err
	}
	mathAgent, err := buildMathAgent(ctx)
	if err != nil {
		return nil, err
	}

	return supervisor.New(ctx, &supervisor.Config{
		//主agent
		Supervisor: sv,
		//子agents配置
		SubAgents: []adk.Agent{searchAgent, mathAgent},
	})
}

func Agent() {
	ctx := context.Background()
	//增加链路追踪
	traceCloseFn, startSpanFn := trace.AppendCozeLoopCallbackIfConfigured(ctx)
	defer traceCloseFn(ctx)
	//构建agent
	sv, err := buildSupervisor(ctx)
	if err != nil {
		log.Fatalf("build supervisor failed: %v", err)
	}

	query := "计算2024年美国和纽约州的国内生产总值。纽约州占美国GDP的百分比是多少?"

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           sv,
		EnableStreaming: true,
	})

	ctx, endSpanFn := startSpanFn(ctx, "Supervisor", query)

	iter := runner.Query(ctx, query)

	fmt.Println("\nuser query: ", query)

	var lastMessage adk.Message
	for {
		event, hasEvent := iter.Next()
		if !hasEvent {
			break
		}

		prints.Event(event)

		if event.Output != nil {
			lastMessage, _, err = adk.GetMessage(event)
		}
	}

	endSpanFn(ctx, lastMessage)

	// wait for all span to be ended
	time.Sleep(5 * time.Second)
}

func init() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}
}
