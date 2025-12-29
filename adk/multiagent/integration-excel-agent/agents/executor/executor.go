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

package executor

import (
	"context"
	"likeeino/adk/multiagent/integration-excel-agent/utils"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// 执行器提示词模版
var executorPrompt = prompt.FromMessages(schema.FString,
	schema.SystemMessage(`你是一个勤奋细致的执行代理人。遵循既定计划，仔细彻底地执行任务。

Available Tools:
- CodeAgent: 此工具是专门用于Excel文件处理的代码代理。它采取循序渐进的计划，通过生成Python代码（利用pandas进行数据分析/操作，利用matplotlib进行绘图/可视化，利用openpyxl进行Excel读/写）来处理每个任务，并按顺序执行任务。当需要对Excel操作进行逐步Python编码时，React代理应该调用它，以确保精确、高效地完成任务。

Notice:
- 不要转移到其他代理，只能使用工具。
`),
	schema.UserMessage(`## OBJECTIVE
{input}
## Given the following plan:
{plan}
## COMPLETED STEPS & RESULTS
{executed_steps}
## Your task is to execute the first step, which is: 
{step}`))

func NewExecutor(ctx context.Context, operator commandline.Operator) (adk.Agent, error) {
	//构建大模型
	cm, err := utils.NewChatModel(ctx,
		utils.WithMaxTokens(4096),
		utils.WithTemperature(float32(0)),
		utils.WithTopP(float32(0)),
	)
	if err != nil {
		return nil, err
	}

	//代码生成代理
	ca, err := newCodeAgent(ctx, operator)
	if err != nil {
		return nil, err
	}

	//web搜索agent
	sa, err := newWebSearchAgent(ctx)
	if err != nil {
		return nil, err
	}

	a, err := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: cm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					//执行器将该代理作为工具进行使用
					adk.NewAgentTool(ctx, ca),
					adk.NewAgentTool(ctx, sa),
				},
			},
		},
		MaxIterations: 20,
		//执行器输入生成函数
		GenInputFn: func(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
			planContent, err := in.Plan.MarshalJSON()
			if err != nil {
				return nil, err
			}

			return executorPrompt.Format(ctx, map[string]any{
				"input":          utils.FormatInput(in.UserInput),
				"plan":           string(planContent),
				"executed_steps": utils.FormatExecutedSteps(in.ExecutedSteps),
				"step":           in.Plan.FirstStep(),
			})
		},
	})
	if err != nil {
		return nil, err
	}

	return a, nil
}
