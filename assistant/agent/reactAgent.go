package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/callbacks"
	t "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/coze-dev/cozeloop-go"
	"github.com/joho/godotenv"
	"io"
	"likeeino/pkg/tool"
	"log"
	"os"
	"time"
)

func ReactAgent() {
	startTime := time.Now()
	fmt.Printf("程序开始执行时间: %s\n", startTime.Format("2006-01-02 15:04:05.000"))

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

	arkModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  os.Getenv("OPENAI_MODEL_NAME"),
	})

	if err != nil {
		fmt.Printf("failed to create chat model: %v", err)
		return
	}
	addtool := tool.GetAddTool()
	subtool := tool.GetSubTool()
	analyzetool := tool.GetAnalyzeTool()
	persona := `#Character:
	你是一个幼儿园老师，会同时判断题目难易程度，给出问题的答案
	`
	// toolCallChecker 用于检查从流中读取的消息是否包含工具调用。
	// 它会持续从流中接收消息，直到找到一个包含工具调用的消息或流结束。
	toolCallChecker := func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
		// 确保在函数退出时关闭流读取器，以防资源泄漏。
		defer sr.Close()
		// 无限循环，用于持续从流中读取消息。
		for {
			// 从流中接收一条消息。如果流中没有新消息，此调用会阻塞。
			msg, err := sr.Recv()
			// 检查接收过程中是否发生错误。
			if err != nil {
				// 如果错误是 io.EOF，表示流已正常结束，没有更多消息了。
				if errors.Is(err, io.EOF) {
					// 正常结束，跳出循环。
					break
				}

				// 如果是其他类型的错误，则直接返回错误。
				return false, err
			}

			// 检查收到的消息是否包含任何工具调用。
			if len(msg.ToolCalls) > 0 {
				// 如果找到工具调用，立即返回 true，表示检查成功。
				return true, nil
			}
		}
		// 如果完整遍历了流而没有找到任何工具调用，则返回 false。
		return false, nil
	}
	raAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: arkModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools:               []t.BaseTool{addtool, subtool, analyzetool},
			ExecuteSequentially: false,
		},
		StreamToolCallChecker: toolCallChecker,
	})
	if err != nil {
		fmt.Printf("failed to create agent: %v", err)
		return
	}
	//构建输入模板schema
	chatmsg := []*schema.Message{
		{
			Role:    schema.System,
			Content: persona,
		},
		{
			Role:    schema.User,
			Content: "请同时告诉我183+192-90这道题的难易程度和答案",
		},
	}
	//流式调用
	//添加了loggerCallback的回调函数
	sr, err := raAgent.Stream(ctx, chatmsg, agent.WithComposeOptions(compose.WithCallbacks(&loggerCallback{})))
	if err != nil {
		fmt.Printf("failed to stream: %v", err)
		return
	}
	//构建最终回答
	finalContent := ""
	for {
		msg, err := sr.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// finish
				break
			}
			fmt.Printf("failed to recv: %v", err)
			return
		}
		finalContent += msg.Content
		//fmt.Printf("%v", msg.Content)
	}

	fmt.Printf("\n\n===== final answer =====\n\n")
	fmt.Printf("%s", finalContent)
	fmt.Printf("\n\n===== finished =====\n")

	// 计算并打印运行时间
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	fmt.Printf("\n程序结束执行时间: %s\n", endTime.Format("2006-01-02 15:04:05.000"))
	fmt.Printf("总运行时间: %v\n", duration)
	fmt.Printf("总运行时间(毫秒): %.2f ms\n", float64(duration.Nanoseconds())/1000000)
}

type loggerCallback struct {
	callbacks.HandlerBuilder
}

func (cb *loggerCallback) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	fmt.Println("==================" + info.Name)
	inputStr, _ := json.MarshalIndent(input, "", "  ") // nolint: byted_s_returned_err_check
	fmt.Printf("[OnStart] %s\n", string(inputStr))
	return ctx
}

func (cb *loggerCallback) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	fmt.Println("=========[OnEnd]=========" + info.Name)
	outputStr, _ := json.MarshalIndent(output, "", "  ") // nolint: byted_s_returned_err_check
	fmt.Println(string(outputStr))
	return ctx
}

func (cb *loggerCallback) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	fmt.Println("=========[OnError]=========" + info.Name)
	fmt.Println(err)
	return ctx
}

func (cb *loggerCallback) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
	output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {

	var graphInfoName = react.GraphName

	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("[OnEndStream] panic err:", err)
			}
		}()

		defer output.Close() // remember to close the stream in defer

		fmt.Println("=========[OnEndStream]=========")
		for {
			frame, err := output.Recv()
			if errors.Is(err, io.EOF) {
				// finish
				break
			}
			if err != nil {
				fmt.Printf("internal error: %s\n", err)
				return
			}

			s, err := json.Marshal(frame)
			if err != nil {
				fmt.Printf("internal error: %s\n", err)
				return
			}

			if info.Name == graphInfoName { // 仅打印 graph 的输出, 否则每个 stream 节点的输出都会打印一遍
				fmt.Printf("%s: %s\n", info.Name, string(s))
			}
		}

	}()
	return ctx
}

func (cb *loggerCallback) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo,
	input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	defer input.Close()
	return ctx
}

func init() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}
}
