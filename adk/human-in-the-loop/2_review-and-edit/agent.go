/**

 */

package main

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
	"likeeino/adk/common/model"
	"likeeino/internal/logs"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	tool2 "likeeino/adk/common/tool"
)

func NewTicketAgent() adk.Agent {
	ctx := context.Background()
	// 注册为全局 handler，这样后续的工具节点都会触发
	callbacks.AppendGlobalHandlers(&loggerCallbacks{})

	a, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "TicketBooker",
		Description: "An agent that can book tickets",
		Instruction: `You are an expert ticket booker. Your goal is to book a ticket for a user.
If you have enough information (e.g., location, passenger name, etc.), use the 'BookTicket' tool to book the ticket.
If you are missing information, you should ask the user for it.`,
		Model: model.NewChatModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{&tool2.InvokableReviewEditTool{InvokableTool: NewBookTicketTool()}},
			},
		},
	})
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create chatmodel: %w", err))
	}

	return a
}

// ------------------
type loggerCallbacks struct{}

func (l *loggerCallbacks) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	logs.Infof("start name: %v, type: %v, component: %v, input: %v", info.Name, info.Type, info.Component, input)
	return ctx
}

func (l *loggerCallbacks) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	logs.Infof("end name: %v, type: %v, component: %v, output: %v", info.Name, info.Type, info.Component, output)
	return ctx
}

func (l *loggerCallbacks) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	logs.Infof("error name: %v, type: %v, component: %v, error: %v", info.Name, info.Type, info.Component, err)
	return ctx
}

func (l *loggerCallbacks) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	return ctx
}

func (l *loggerCallbacks) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	return ctx
}
