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

package replanner

import (
	"context"
	"fmt"
	"likeeino/adk/multiagent/integration-excel-agent/agents"
	"likeeino/adk/multiagent/integration-excel-agent/generic"
	"likeeino/adk/multiagent/integration-excel-agent/params"
	"likeeino/adk/multiagent/integration-excel-agent/tools"
	"likeeino/adk/multiagent/integration-excel-agent/utils"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

var (
	replannerPromptTemplate = prompt.FromMessages(schema.Jinja2,
		schema.SystemMessage(`您是专门从事Excel数据处理任务的专家规划师。你的目标是了解用户需求，并将其分解为一个清晰的、循序渐进的计划

**1.理解目标:**
- 仔细分析用户的请求，以确定最终目标。
- 识别输入数据（Excel文件）和所需的输出格式。

**2. 交付物:**
- 最终输出应该是一个表示计划的JSON对象，其中包含一系列步骤。
- 对于将执行此步骤的代理，每个步骤都必须是清晰简洁的说明。

**3. Plan Decomposition Principles:**
- **粒度：**将任务分解为尽可能小的逻辑步骤。例如，不要“处理数据”，而是使用“读取Excel文件”、“过滤掉缺少值的行”、“计算“销售额”列的平均值”等。
- **顺序：**步骤应按照正确的执行顺序进行。
- **清晰度：**每个步骤都应该是明确的，并且易于执行此步骤的代理理解。
**4. Output Format (Few-shot Example):**
	Here is an example of a good plan:
User Request: "Please calculate the average sales for each product category in the attached 'sales_data.xlsx' file and generate a report."
{
  "steps": [
    {
      "instruction": "Read the 'sales_data.xlsx' file into a pandas DataFrame."
    },
    {
      "instruction": "Group the DataFrame by 'Product Category' and calculate the mean of the 'Sales' column for each group."
    },
    {
      "instruction": "Summarize the average sales for each product category and present the results in a table."
    }
  ]
}

**5. 限制:**
- 不要直接在计划中生成代码。
- 确保该计划合乎逻辑且可实现。
- 最后一步应该始终是生成报告或提供最终结果。

**6. 重新规划:**
- 如果当前计划已完成，请调用“submit_result”工具。
- 如果需要修改或扩展计划，请使用新计划调用“create_plan”工具。
`),
		schema.UserMessage(`
User Query: {{ user_query }}
Current Time: {{ current_time }}
File Preview:
{{ file_preview }}
Executed Steps: {{ executed_steps }}
Remaining Steps: {{ remaining_steps }}
`),
	)
)

// NewReplanner Plan-Execute 中的重规划器
func NewReplanner(ctx context.Context, op commandline.Operator) (adk.Agent, error) {
	cm, err := utils.NewChatModel(ctx,
		utils.WithMaxTokens(4096),
		utils.WithTopP(0),
		utils.WithTemperature(1.0),
		utils.WithDisableThinking(true),
	)
	if err != nil {
		return nil, err
	}

	respondInfo, err := tools.NewToolSubmitResult(op).Info(context.Background())
	if err != nil {
		return nil, err
	}

	//创建重规划器agent
	a, err := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel:   cm,
		PlanTool:    generic.PlanToolInfo,
		GenInputFn:  replannerInputGen, //生成输入消息
		RespondTool: respondInfo,
		NewPlan: func(ctx context.Context) planexecute.Plan {
			return &generic.Plan{}
		},
	})
	if err != nil {
		return nil, err
	}

	return agents.NewWrite2PlanMDWrapper(a, op), nil
}

func replannerInputGen(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
	pf, _ := params.GetTypedContextParams[string](ctx, params.UserAllPreviewFilesSessionKey)
	plan, ok := in.Plan.(*generic.Plan)
	if !ok {
		return nil, fmt.Errorf("plan is not Plan type")
	}

	// remove the first step
	//每次执行都移除第一步: 因为每次执行都会生成一个步骤，所以每次执行都移除第一步
	plan.Steps = plan.Steps[1:]
	planStr, err := sonic.MarshalString(plan)
	if err != nil {
		return nil, err
	}

	userInput, err := sonic.MarshalString(in.UserInput)
	if err != nil {
		return nil, err
	}

	return replannerPromptTemplate.Format(ctx, map[string]any{
		"current_time":    utils.GetCurrentTime(),
		"file_preview":    pf,
		"user_query":      userInput,
		"remaining_steps": planStr,
		"executed_steps":  utils.FormatExecutedSteps(in.ExecutedSteps),
	})
}
