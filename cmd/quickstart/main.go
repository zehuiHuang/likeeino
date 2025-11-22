package main

import (
	"context"
	"likeeino/chat"
	"log"

	"github.com/joho/godotenv"
)

func main() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}

	ctx := context.Background()

	// 使用模版创建messages
	log.Printf("===create messages===\n")
	messages := chat.CreateMessagesFromTemplate()
	log.Printf("messages: %+v\n\n", messages)

	// 创建llm
	log.Printf("===create llm===\n")
	cm := chat.CreateDeepSeekChatModel(ctx)
	// cm := chat.CreateOllamaChatModel(ctx)
	log.Printf("create llm success\n\n")

	log.Printf("===llm generate===\n")
	result := chat.Generate(ctx, cm, messages)
	log.Printf("result: %+v\n\n", result)

	log.Printf("===llm stream generate===\n")
	streamResult := chat.Stream(ctx, cm, messages)
	chat.ReportStream(streamResult)

}
