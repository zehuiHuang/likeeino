/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package model

import (
	"context"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	arkModel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"log"
	"os"
	"strings"
)

func NewChatModel(tp ...string) model.ToolCallingChatModel {
	modelType := strings.ToLower(os.Getenv("MODEL_TYPE"))
	if len(tp) > 0 {
		modelType = tp[0]
	}
	//httpClient := &http.Client{
	//	Timeout: 60 * time.Second, // 设置超时时间
	//}
	if modelType == "deepseek" {
		cm, err := deepseek.NewChatModel(context.Background(), &deepseek.ChatModelConfig{
			APIKey: os.Getenv("OPENAI_API_KEY"),
			Model:  os.Getenv("OPENAI_MODEL_NAME"),
		})
		if err != nil {
			log.Fatalf("ark.NewChatModel failed: %v", err)
		}
		return cm
	}

	// Create Ark ChatModel when MODEL_TYPE is "ark"
	if modelType == "ark" {
		cm, err := ark.NewChatModel(context.Background(), &ark.ChatModelConfig{
			// Add Ark-specific configuration from environment variables
			APIKey: os.Getenv("ARK_API_KEY"),
			Model:  os.Getenv("ARK_MODEL"),
			//BaseURL: os.Getenv("ARK_BASE_URL"),
			Thinking: &arkModel.Thinking{
				Type: arkModel.ThinkingTypeDisabled,
			},
		})
		if err != nil {
			log.Fatalf("ark.NewChatModel failed: %v", err)
		}
		return cm
	}

	// Create OpenAI ChatModel (default)
	cm, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   os.Getenv("OPENAI_MODEL"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		ByAzure: func() bool {
			return os.Getenv("OPENAI_BY_AZURE") == "true"
		}(),
	})
	if err != nil {
		log.Fatalf("openai.NewChatModel failed: %v", err)
	}
	return cm
}
