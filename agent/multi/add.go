package multi

import (
	"context"
	"fmt"
	"likeeino/pkg/tool"
	"os"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	t "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/multiagent/host"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

func newAddSpecialist(ctx context.Context) (*host.Specialist, error) {

	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  os.Getenv("OPENAI_MODEL_NAME"),
	})
	if err != nil {
		fmt.Printf("failed to create chat model: %v", err)
		return nil, err
	}
	addtool := tool.GetAddTool()
	raAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []t.BaseTool{addtool},
		},
	})
	if err != nil {
		return nil, err
	}
	return &host.Specialist{
		AgentMeta: host.AgentMeta{
			Name:        "add_specialist",
			IntendedUse: "add two numbers and return the result",
		},
		Invokable: func(ctx context.Context, input []*schema.Message, opts ...agent.AgentOption) (*schema.Message, error) {
			return raAgent.Generate(ctx, input, opts...)
		},
	}, nil
}
