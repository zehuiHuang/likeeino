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

package agents

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

func NewResearchAgent(ctx context.Context, tcm model.ToolCallingChatModel) (adk.Agent, error) {
	type webSearchInput struct {
		CurrentContext string `json:"current_context" jsonschema_description:"current context for web search"`
	}
	type webSearchOutput struct {
		Result []string
	}
	webSearchTool, err := utils.InferTool(
		"web_search",
		"web search tool",
		func(ctx context.Context, input *webSearchInput) (output *webSearchOutput, err error) {
			// replace it with real web search tool
			if input.CurrentContext == "" {
				return nil, fmt.Errorf("web search input is required")
			}
			return &webSearchOutput{}, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ResearchAgent",
		Description: "ResearchAgent负责进行研究并生成可行的解决方案。它支持中断从用户那里接收额外的上下文信息，这有助于提高研究结果的准确性和相关性。它利用网络搜索工具收集最新信息。",
		Instruction: `你是研究代理人。你的角色是:

	- 对给定的主题或问题进行彻底的研究。
	- 根据您的发现，制定可行且充分知情的解决方案。
	-通过随时接受用户提供的其他上下文或信息来改进您的研究，从而支持中断。
	-有效地使用网络搜索工具来收集相关和最新的数据。
	-清晰、逻辑清晰地传达你的研究结果。
	-如果需要提高研究质量，可以提出澄清问题。
	-在整个互动过程中保持专业和乐于助人的语气。
	
	Tool Handling:
	-当您认为输入信息不足以支持研究时，请使用ask_for_clarification工具要求用户补充上下文。
	-如果上下文满足，您可以使用web_search工具从互联网获取更多数据。
	`,
		Model: tcm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{webSearchTool, newAskForClarificationTool()},
			},
		},
		MaxIterations: 5,
	})
}

type askForClarificationOptions struct {
	NewInput *string
}

func WithNewInput(input string) tool.Option {
	return tool.WrapImplSpecificOptFn(func(t *askForClarificationOptions) {
		t.NewInput = &input
	})
}

type AskForClarificationInput struct {
	Question string `json:"question" jsonschema_description:"The specific question you want to ask the user to get the missing information"`
}

func newAskForClarificationTool() tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"ask_for_clarification",
		"当用户的请求不明确或缺乏继续进行所需的信息时，调用此工具。在你有效使用其他工具之前，用它来问一个后续问题，以获得你需要的细节，比如这本书的类型。",
		func(ctx context.Context, input *AskForClarificationInput, opts ...tool.Option) (output string, err error) {
			o := tool.GetImplSpecificOptions[askForClarificationOptions](nil, opts...)
			if o.NewInput == nil {
				return "", compose.Interrupt(ctx, input.Question)
			}
			output = *o.NewInput
			o.NewInput = nil
			return output, nil
		})
	if err != nil {
		log.Fatal(err)
	}
	return t
}
