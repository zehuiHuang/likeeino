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

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

func NewCodeAgent(ctx context.Context, tcm model.ToolCallingChatModel) (adk.Agent, error) {
	type RAGInput struct {
		Query   string  `json:"query" jsonschema_description:"query for search"`
		Context *string `json:"context" jsonschema_description:"user input context"`
	}
	type RAGOutput struct {
		Documents []string `json:"documents"`
	}
	knowledgeBaseTool, err := utils.InferTool(
		"knowledge_base",
		"知识库，可以回答常见问题，提供答案的具体原因，并提高准确性",
		func(ctx context.Context, input *RAGInput) (output *RAGOutput, err error) {
			// replace it with real knowledge base search
			if input.Query == "" {
				return nil, fmt.Errorf("RAG Input query is required")
			}
			return &RAGOutput{
				[]string{
					"Q： Python中列表和元组有什么区别？\nA：列表是可变的，这意味着您可以在创建后修改其元素，而元组是不可变的，一旦创建就不能更改。列表使用方括号[]，元组使用括号（）。",
					"Q： Java中如何处理异常？\nA：在Java中，您可以使用try-catch块来处理异常。可能引发异常的代码被放置在try块中，catch块处理异常。可选地，可以使用finally块进行清理。",
					"Q： JavaScript中async和wait关键字的用途是什么？\nA:async将函数标记为异步，允许它返回Promise。wait会暂停异步函数的执行，直到Promise解析，从而使异步代码编写更容易。",
					"Q： 如何优化SQL查询以获得更好的性能？\nA：常见的优化包括在频繁查询的列上创建索引、避免SELECT*、高效使用JOIN以及分析查询执行计划以识别瓶颈。",
					"Q： 什么是依赖注入，为什么它有用？\nA：依赖注入是一种设计模式，其中对象从外部源接收其依赖关系，而不是自己创建它们。它促进了松耦合、更容易的测试和更好的代码可维护性。",
				},
			}, nil
		})
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "CodeAgent",
		Description: "The CodeAgent specializes in generating high-quality code by leveraging a knowledge base as a tool. It recalls relevant knowledge and best practices to produce efficient, maintainable, and accurate code solutions tailored to the project requirements.",
		Instruction: `You are the CodeAgent. Your responsibilities include:

- Generating high-quality, efficient, and maintainable code based on the project requirements.
- Utilizing a knowledge base tool to recall relevant coding standards, patterns, and best practices.
- Ensuring the code is clear, well-documented, and meets the specified functionality.
- Reviewing related knowledge to enhance the accuracy and quality of your code.
- Communicating your coding decisions and providing explanations when necessary.
- Responding promptly and professionally to user requests or clarifications.

Tool handling:
When the user's question is vague or exceeds the scope of your answer, please use the knowledge_base tool to recall relevant results from the knowledge base and provide accurate answers based on the results.
`,
		Model: tcm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{knowledgeBaseTool},
			},
		},
		MaxIterations: 3,
	})
}
