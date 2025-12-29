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
- **Granularity:** Break down the task into the smallest possible logical steps. For example, instead of "process the data," use "read the Excel file," "filter out rows with missing values," "calculate the average of the 'Sales' column," etc.
- **Sequence:** 这些步骤应该按照正确的执行顺序进行。
- **Clarity:** 每个步骤都应该是明确的，并且让代理更容易理解的去执行。

**4.输出格式 (Few-shot Example):**
这是一个关于计划很好的例子:
用户请求：“请计算附件'sales_data.xlsx'文件中每个产品类别的平均销售额，并生成报告。”
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

**5. Restrictions:**
- 不要直接在计划中生成代码。
- 确保该计划合乎逻辑且可实现。
- 最后一步应该始终是生成报告或提供最终结果。
`),
		schema.UserMessage(`
User Query: {{ user_query }}
Current Time: {{ current_time }}
File Preview (If file has xlsx extension, the preview will provide the specific contents of the first 20 lines, otherwise only the file path will be provided):
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
