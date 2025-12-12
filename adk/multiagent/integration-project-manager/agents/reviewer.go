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

import "github.com/cloudwego/eino/adk"

/**
workflowAgents:
允许开发者以预设的流程来组织和执行多个子 Agent
*/
import (
	"context"

	"github.com/cloudwego/eino/components/model"
)

func NewReviewAgent(ctx context.Context, tcm model.ToolCallingChatModel) (adk.Agent, error) {
	// these sub-agents don't need description because they'll be set in a fixed workflow.
	questionAnalysisAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "question_analysis_agent",
		Description: "问题分析代理",
		Instruction: `您是问题分析代理。您的职责包括：

	-分析给定的研究或编码结果，以确定关键问题和评估标准。
	-将复杂问题分解为清晰、可管理的组件。
	-突出潜在问题或关注领域。
	-准备一个结构化的框架来指导后续的审查生成。
	-在传递内容之前，确保对内容有深入的理解。`,
		Model: tcm,
	})
	if err != nil {
		return nil, err
	}

	generateReviewAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "generate_review_agent",
		Description: "生成审核代理",
		Instruction: `您是生成审核代理。你的职责是：

	- 根据问题分析进行全面和平衡的评论。
	- 突出优势、劣势和需要改进的地方。
	- 提供建设性和可操作的反馈。
	- 保持评估的客观性和清晰度。
	- 准备审核内容，以便下一步进行验证。`,
		Model: tcm,
	})
	if err != nil {
		return nil, err
	}

	reviewValidationAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "review_validation_agent",
		Description: "审查验证代理",
		Instruction: `您是审核验证代理。您的任务是：

	- 验证生成的审查的准确性、连贯性和公平性。
	- 检查逻辑的一致性和完整性。
	- 识别任何偏差或错误，并提出纠正建议。
	- 确认审查与原始分析和项目目标一致。
	- 批准最终演示文稿的审查，或在必要时要求修改。`,
		Model: tcm,
	})
	if err != nil {
		return nil, err
	}

	return adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        "ReviewAgent",
		Description: "ReviewAgent负责通过顺序工作流评估研究和编码结果。它协调了三个关键步骤——问题分析、评审生成和评审验证——以提供合理的评估，支持项目管理决策。",
		SubAgents:   []adk.Agent{questionAnalysisAgent, generateReviewAgent, reviewValidationAgent},
	})
}
