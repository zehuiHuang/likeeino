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
	"fmt"
	"likeeino/adk/multiagent/integration-excel-agent/params"
	"likeeino/adk/multiagent/integration-excel-agent/tools"
	"likeeino/adk/multiagent/integration-excel-agent/utils"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func newCodeAgent(ctx context.Context, operator commandline.Operator) (adk.Agent, error) {
	cm, err := utils.NewChatModel(ctx,
		utils.WithMaxTokens(12000),
		utils.WithTemperature(float32(1)),
		utils.WithTopP(float32(1)),
	)
	if err != nil {
		return nil, err
	}

	preprocess := []tools.ToolRequestPreprocess{tools.ToolRequestRepairJSON}

	ca, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name: "CodeAgent",
		Description: `此子代理是专门处理Excel文件的代码代理. 
它接收一个明确的任务，并通过生成Python代码来完成任务并执行它。
该代理利用pandas进行数据分析和操作，利用matplotlib进行绘图和可视化，利用openpyxl读取和写入Excel文件。
每当需要对Excel文件操作进行逐步Python编码时，React代理都应该调用此子代理，以确保精确高效的任务执行。
`,
		Instruction: `Y你是一名代码代理。您的工作流程如下:
1.您将获得一个明确的任务来处理Excel文件。
2.你应该分析任务并使用正确的工具来帮助编码。
3.你应该编写python代码来完成任务。
4.您最好将代码执行结果写入另一个文件以供进一步使用。

您处于 react mode, 并且应该使用以下库来帮助您完成任务：:
- pandas：用于数据分析和操作
- matplotlib：用于绘图和可视化
- openpyxl：用于读取和写入Excel文件

Notice:
1. Tool Calls参数必须是有效的json。
2. 工具调用参数不应包含无效后缀，如“]<|FunctionCallEnd|>”。
`,
		Model: cm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					//五个工具,并且增加了预处理和后处理,让输入和输出更规范
					tools.NewWrapTool(tools.NewBashTool(operator), preprocess, []tools.ToolResponsePostprocess{tools.FilePostProcess}),
					tools.NewWrapTool(tools.NewTreeTool(operator), preprocess, nil),
					tools.NewWrapTool(tools.NewEditFileTool(operator), preprocess, []tools.ToolResponsePostprocess{tools.EditFilePostProcess}),
					tools.NewWrapTool(tools.NewReadFileTool(operator), preprocess, nil), // TODO: compress post process
					tools.NewWrapTool(tools.NewPythonRunnerTool(operator), preprocess, []tools.ToolResponsePostprocess{tools.FilePostProcess}),
				},
			},
		},
		//执行器输入生成函数
		GenModelInput: func(ctx context.Context, instruction string, input *adk.AgentInput) ([]adk.Message, error) {
			wd, ok := params.GetTypedContextParams[string](ctx, params.WorkDirSessionKey)
			if !ok {
				return nil, fmt.Errorf("work dir not found")
			}

			tpl := prompt.FromMessages(schema.Jinja2,
				schema.SystemMessage(instruction),
				schema.UserMessage(`WorkingDirectory: {{ working_dir }}
UserQuery: {{ user_query }}
CurrentTime: {{ current_time }}
`))

			msgs, err := tpl.Format(ctx, map[string]any{
				"working_dir":  wd,
				"user_query":   utils.FormatInput(input.Messages),
				"current_time": utils.GetCurrentTime(),
			})
			if err != nil {
				return nil, err
			}

			return msgs, nil
		},
		OutputKey:     "",
		MaxIterations: 1000,
	})
	if err != nil {
		return nil, err
	}

	return ca, nil
}
