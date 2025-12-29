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

package report

import (
	"context"
	"encoding/json"
	"fmt"
	"likeeino/adk/multiagent/integration-excel-agent/generic"
	"likeeino/adk/multiagent/integration-excel-agent/params"
	"likeeino/adk/multiagent/integration-excel-agent/tools"
	"likeeino/adk/multiagent/integration-excel-agent/utils"
	"os"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// NewReportAgent 报告agent
func NewReportAgent(ctx context.Context, operator commandline.Operator) (adk.Agent, error) {
	cm, err := utils.NewChatModel(ctx,
		utils.WithMaxTokens(12000),
		utils.WithTemperature(0.1),
		utils.WithTopP(1),
	)
	if err != nil {
		return nil, err
	}

	var imageReaderTool tool.InvokableTool
	//视觉大模型配置,默认不开启
	if modelName := os.Getenv("ARK_VISION_MODEL"); modelName != "" {
		visionModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
			APIKey:  os.Getenv("ARK_VISION_API_KEY"),
			BaseURL: os.Getenv("ARK_VISION_BASE_URL"),
			Region:  os.Getenv("ARK_VISION_REGION"),
			Model:   modelName,
		})
		if err != nil {
			return nil, err
		}
		imageReaderTool = tools.NewToolImageReader(visionModel)
	}
	//ToolRequestRepairJSON:修复和纠正可能不规范的JSON格式(预处理)
	preprocess := []tools.ToolRequestPreprocess{tools.ToolRequestRepairJSON}
	agentTools := []tool.BaseTool{
		//1、NewWrapTool:包装类(函数增强),把函数的输入和输出进行JSON格式化
		//2、NewBashTool、NewTreeTool等工具都传入operator参数(接口),代码逻辑执行实际的操作
		//相当于把工具的能力放到到接口实现的参数上了
		tools.NewWrapTool(tools.NewBashTool(operator), preprocess, nil),
		tools.NewWrapTool(tools.NewTreeTool(operator), preprocess, nil),
		tools.NewWrapTool(tools.NewEditFileTool(operator), preprocess, nil),
		tools.NewWrapTool(tools.NewReadFileTool(operator), preprocess, nil),
		tools.NewWrapTool(tools.NewToolSubmitResult(operator), preprocess, nil),
	}
	if imageReaderTool != nil {
		agentTools = append(agentTools, tools.NewWrapTool(imageReaderTool, preprocess, nil))
	}

	ra, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name: "Report",
		Description: `
这是一个报告代理，负责从给定的file_path读取文件，并根据其内容生成全面的报告。
其工作流程包括读取文件、分析数据和信息、总结关键发现和见解，以及生成一份清晰简洁的报告来解决用户的查询。
如果文件包含图表或可视化，代理将在报告中适当地引用它们。当需要从指定文件生成详细的数据驱动报告时，React代理应调用此子代理。`,
		Instruction: `你是一名报告代理人。您的任务是读取给定file_path处的文件，并根据其内容生成一份全面的报告。

**Workflow:**
1.读取由“输入文件路径”和“工作目录”指定的文件内容。
2.分析文件中的数据和信息。
3.总结主要发现和见解。
4.生成一份清晰简洁的报告，回答用户的询问。
5.如果有任何图表或可视化，请在报告中参考。
6.如果工作已经完成，必须在完成之前调用SubmitResult工具。
`,
		Model: cm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: agentTools,
			},
			//当代理在调用时立即返回的工具???
			ReturnDirectly: tools.SubmitResultReturnDirectly,
		},
		GenModelInput: func(ctx context.Context, instruction string, input *adk.AgentInput) ([]adk.Message, error) {
			planExecuteResult := input.Messages
			if len(input.Messages) > 0 && input.Messages[len(input.Messages)-1].Role == schema.Tool {
				planExecuteResult = []*schema.Message{input.Messages[len(input.Messages)-1]}
			}

			fp, ok := params.GetTypedContextParams[string](ctx, params.FilePathSessionKey)
			if !ok {
				return nil, fmt.Errorf("file path session key not found")
			}

			plan, ok := utils.GetSessionValue[*generic.Plan](ctx, planexecute.PlanSessionKey)
			if !ok {
				return nil, fmt.Errorf("plan not found")
			}

			planStr, err := json.MarshalIndent(plan, "", "\t")
			if err != nil {
				return nil, err
			}

			wd, ok := params.GetTypedContextParams[string](ctx, params.WorkDirSessionKey)
			if !ok {
				return nil, fmt.Errorf("work dir not found")
			}

			files, err := generic.ListDir(wd)
			if err != nil {
				return nil, err
			}

			tpl := prompt.FromMessages(schema.Jinja2,
				schema.SystemMessage(instruction),
				schema.UserMessage(`
User Query: {{ user_query }}
Input File Path: {{ file_path }}
Working Directory: {{ work_dir }}
Working Directory Files: {{ work_dir_files }}
Current Time: {{ current_time }}

**Plan Details:**
{{ plan }}
`))

			msgs, err := tpl.Format(ctx, map[string]any{
				"file_path":      fp,
				"work_dir":       wd,
				"work_dir_files": utils.ToJSONString(files),
				"user_query":     utils.FormatInput(planExecuteResult),
				"plan":           string(planStr),
				"current_time":   utils.GetCurrentTime(),
			})
			if err != nil {
				return nil, err
			}

			return msgs, nil
		},
		MaxIterations: 20,
	})
	if err != nil {
		return nil, err
	}

	return ra, nil
}
