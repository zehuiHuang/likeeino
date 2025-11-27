package multi

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/flow/agent/multiagent/host"
)

func newHost(ctx context.Context) (*host.Host, error) {
	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  os.Getenv("OPENAI_MODEL_NAME"),
	})
	if err != nil {
		fmt.Printf("failed to create chat model: %v", err)
		return nil, err
	}

	return &host.Host{
		ToolCallingModel: chatModel,
		SystemPrompt:     "你可以同时计算加法和减法。当用户提问时，你需要回答问题。",
	}, nil
}
