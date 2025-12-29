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

package planner

import (
	"context"
	"fmt"
	"likeeino/adk/multiagent/integration-excel-agent/agents"
	"likeeino/adk/multiagent/integration-excel-agent/generic"
	"likeeino/adk/multiagent/integration-excel-agent/params"
	"likeeino/adk/multiagent/integration-excel-agent/utils"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

var (
	plannerPromptTemplate = prompt.FromMessages(schema.Jinja2,
		schema.SystemMessage(`您是专门从事Excel数据处理任务的专家规划师。你的目标是了解用户需求，并将其分解为一个清晰的、循序渐进的计划。

**1. 理解目标:**
- 仔细分析用户的请求，以确定最终目标。
- 识别输入数据（Excel文件）和所需的输出格式。

**2. 交付物:**
- 最终输出应该是一个表示计划的JSON对象，其中包含一系列步骤。
- 对于将执行此步骤的代理，每个步骤都必须是清晰简洁的说明。

**3. Plan Decomposition Principles:**
- **Granularity:** 将任务分解为尽可能小的逻辑步骤。例如，不要“处理数据”，而是使用“读取Excel文件”、“过滤掉缺少值的行”、“计算“销售额”列的平均值”等。
- **Sequence:** 这些步骤应该按照正确的执行顺序进行。
- **Clarity:** 每个步骤都应该是明确的，并且让代理更容易理解的去执行。

**4.输出格式 (Few-shot Example):**
这是一个关于计划很好的例子:
用户请求：“请计算附件'sales_data.xlsx'文件中每个产品类别的平均销售额，并生成报告。”
{
  "steps": [
    {
      "instruction": "将“sales_data.xlsx”文件读入pandas DataFrame。"
    },
    {
      "instruction": "按“产品类别”对DataFrame进行分组，并计算每个组的“销售额”列的平均值。"
    },
    {
      "instruction": "总结每个产品类别的平均销售额，并将结果显示在表格中。"
    }
  ]
}

**5. Restrictions:**
- 不要直接在计划中生成代码。
- 确保该计划合乎逻辑且可实现。
- 最后一步应该始终是生成报告或提供最终结果。
`),
		schema.UserMessage(`
User Query: {{ user_query }}
Current Time: {{ current_time }}
File Preview (如果文件扩展名为xlsx，预览将提供前20行的具体内容，否则只提供文件路径):
{{ file_preview }}
`),
	)
)

func NewPlanner(ctx context.Context, op commandline.Operator) (adk.Agent, error) {
	cm, err := newPlannerChatModel(ctx)
	if err != nil {
		return nil, err
	}
	//生成计划器代理
	a, err := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		//指定大模型 进行严格的格式化输出,格式在cm中定义
		ChatModelWithFormattedOutput: cm,
		//为计划器生成输入消息的函数(包括背景描述、用户问题、文件路径、时间等)
		GenInputFn: newPlannerInputGen(plannerPromptTemplate),
		//作用?
		NewPlan: func(ctx context.Context) planexecute.Plan {
			return &generic.Plan{}
		},
	})
	if err != nil {
		return nil, err
	}
	//包装函数
	return agents.NewWrite2PlanMDWrapper(a, op), nil
}

func newPlannerChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	//将自定义的结构ParamsOneOf 转化为model可接受的结构,该结构强制大模型按照定义好的格式进行返回
	sc, err := generic.PlanToolInfo.ToJSONSchema()
	if err != nil {
		return nil, err
	}

	return utils.NewChatModel(ctx,
		utils.WithMaxTokens(4096),
		utils.WithTemperature(0),
		utils.WithTopP(0),
		utils.WithDisableThinking(true),
		utils.WithResponseFormatJsonSchema(&openai.ChatCompletionResponseFormatJSONSchema{
			Name:        generic.PlanToolInfo.Name,
			Description: generic.PlanToolInfo.Desc,
			JSONSchema:  sc,
			Strict:      true,
		}),
	)
}

func newPlannerInputGen(plannerPrompt prompt.ChatTemplate) planexecute.GenPlannerModelInputFn {
	return func(ctx context.Context, userInput []adk.Message) ([]adk.Message, error) {
		pf, _ := params.GetTypedContextParams[string](ctx, params.UserAllPreviewFilesSessionKey)
		fmt.Println("Planner的输入函数newPlannerInputGen的上下文pf:" + pf)
		fmt.Println("Planner的输入函数newPlannerInputGen的参数userInput:" + utils.FormatInput(userInput))
		msgs, err := plannerPrompt.Format(ctx, map[string]any{
			"user_query":   utils.FormatInput(userInput),
			"current_time": utils.GetCurrentTime(),
			"file_preview": pf,
		})
		if err != nil {
			return nil, err
		}

		return msgs, nil
	}
}
