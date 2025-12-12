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
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

func NewProjectManagerAgent(ctx context.Context, tcm model.ToolCallingChatModel) (adk.Agent, error) {
	a, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ProjectManagerAgent",
		Description: "ProjectManagerAgent充当项目工作流的主管和协调员。它根据用户输入和项目需求，动态地路由和协调多个子代理，负责不同维度的工作，如研究、编码和审查。",
		Instruction: `您是ProjectManagerAgent。你的职责是：

	- 监督和协调多个专业的三个子代理：ResearchAgent、CodeAgent、ReviewAgent。
	- ResearchAgent：当您需要进行研究并生成可行的解决方案时，分配此代理。
	- CodeAgent：在需要生成高质量代码时分配此代理。
	- ReviewAgent：当您需要评估研究或编码结果时，分配此代理。
	- 根据当前项目需求，将任务和用户输入动态路由到适当的子代理。
	- 监控每个子代理的进度和产出，以确保与项目目标保持一致。
	- 促进子代理之间的沟通和协作，以优化工作流程效率。
	- 向用户提供有关项目状态和下一步的清晰更新和摘要。
	- 保持专业、有组织和积极主动的项目管理方法。
`,
		Model: tcm,
		Exit:  &adk.ExitTool{},
	})
	if err != nil {
		log.Fatal(err)
	}
	return a, nil
}
