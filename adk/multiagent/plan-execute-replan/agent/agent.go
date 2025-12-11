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

package agent

import (
	"context"
	"fmt"
	"likeeino/adk/common/model"
	"likeeino/adk/multiagent/plan-execute-replan/tools"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func NewPlanner(ctx context.Context) (adk.Agent, error) {
	return planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: model.NewChatModel(),
	})
}

var executorPrompt = prompt.FromMessages(schema.FString,
	schema.SystemMessage(`您是一位勤奋细致的旅行研究执行者，遵循既定计划，认真彻底地执行任务。
	使用可用工具执行每个计划步骤。
	如需查询天气，请使用get_weather工具。
	如需查询航班，请使用搜索航班工具。
	如需搜索酒店，请使用search_hotels工具。
	进行景点研究时，请使用搜索景点工具。
	为用户澄清内容，请使用"ask_for_clarification"工具。总结时需重复问题与结果以确认用户理解，尽量避免打扰用户。
	为每项任务提供详细结果。
	云端调用多个工具以获取最终结果。`),
	schema.UserMessage(`## 目标
{input}
## 考虑到以下计划:
{plan}
## 已完成的步骤和结果
{executed_steps}
## 你的任务是执行第一步，即: 
{step}`))

func formatInput(in []adk.Message) string {
	return in[0].Content
}

func formatExecutedSteps(in []planexecute.ExecutedStep) string {
	var sb strings.Builder
	for idx, m := range in {
		sb.WriteString(fmt.Sprintf("## %d. Step: %v\n  Result: %v\n\n", idx+1, m.Step, m.Result))
	}
	return sb.String()
}

func NewExecutor(ctx context.Context) (adk.Agent, error) {
	// 为执行者准备旅行工具
	travelTools, err := tools.GetAllTravelTools(ctx)
	if err != nil {
		return nil, err
	}
	// 创建执行器
	return planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: model.NewChatModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: travelTools,
			},
		},
		//为执行器生成输入消息。
		GenInputFn: func(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
			planContent, err_ := in.Plan.MarshalJSON()
			fmt.Println("////////////////////////////////////")
			fmt.Printf("GenInputFn参数的输入:%s", planContent)
			fmt.Println("////////////////////////////////////")
			if err_ != nil {
				return nil, err_
			}

			firstStep := in.Plan.FirstStep()

			msgs, err_ := executorPrompt.Format(ctx, map[string]any{
				"input":          formatInput(in.UserInput),
				"plan":           string(planContent),
				"executed_steps": formatExecutedSteps(in.ExecutedSteps),
				"step":           firstStep,
			})
			if err_ != nil {
				return nil, err_
			}

			return msgs, nil
		},
	})
}

func NewRePlanAgent(ctx context.Context) (adk.Agent, error) {
	return planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: model.NewChatModel(),
	})
}
