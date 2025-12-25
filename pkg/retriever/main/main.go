package main

import (
	"context"
	"fmt"
	rr "github.com/cloudwego/eino/components/retriever"
	"github.com/joho/godotenv"
	"likeeino/internal/logs"
	"likeeino/pkg/retriever"
	"log"
)

func main() {
	// 初始化检索器
	ctx := context.Background()
	//rtr, err := retriever.NewRetriever(ctx)
	rtr, err := retriever.NewMilvusRetriever(ctx)
	if err != nil {
		fmt.Println(err)
	} else {
		// 替换为真实的知识库检索
		documents, err := rtr.Retrieve(ctx, "Eino  是什么", rr.WithTopK(3))
		if err != nil {
			fmt.Println(err)
		} else {
			logs.Infof("----------Retrieve query %s, got %d documents---------", "eino是什么", len(documents))
			for _, doc := range documents {
				fmt.Printf("Document ID: %s\nContent: %s\nScore: %f\n\n", doc.ID, doc.Content, doc.Score)
			}
		}
	}
}

func init() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}
}
