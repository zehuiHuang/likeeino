package chat

import (
	"context"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/model"
	"log"
	"os"
)

func CreateDeepSeekChatModel(ctx context.Context) model.ToolCallingChatModel {
	key := os.Getenv("OPENAI_API_KEY")
	modelName := os.Getenv("OPENAI_MODEL_NAME")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
		APIKey:  key,
	})
	if err != nil {
		log.Fatalf("create openai chat model failed, err=%v", err)
	}
	return chatModel
}
