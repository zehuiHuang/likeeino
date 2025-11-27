package multi2

import (
	"context"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/flow/agent/multiagent/host"
)

func newHost(ctx context.Context, baseURL, apiKey, modelName string) (*host.Host, error) {
	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		//BaseURL: baseURL,
		Model:  modelName,
		APIKey: apiKey,
	})
	if err != nil {
		return nil, err
	}

	return &host.Host{
		ToolCallingModel: chatModel,
		SystemPrompt:     "You can read and write journal on behalf of the user. When user asks a question, always answer with journal content.",
	}, nil
}
