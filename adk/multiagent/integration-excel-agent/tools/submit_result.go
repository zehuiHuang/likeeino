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

package tools

import (
	"context"
	"fmt"
	"likeeino/adk/multiagent/integration-excel-agent/generic"
	"likeeino/adk/multiagent/integration-excel-agent/params"
	"likeeino/adk/multiagent/integration-excel-agent/utils"
	"path/filepath"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

var (
	submitResultToolInfo = &schema.ToolInfo{
		Name: "submit_result",
		Desc: "当所有步骤都完成且没有明显问题时，调用此工具结束任务并向用户报告最终执行结果。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"is_success": {
				Type: schema.Boolean,
				Desc: "success or not，true/false",
			},
			"result": {
				Type: schema.String,
				Desc: "Task execution process and result",
			},
			"files": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Desc: `需要交付给用户的最终文件（仅包括最终成功生成的文件，默认情况下不包括Python脚本，除非用户明确要求）。
仅选择能够满足用户原始需求的文档，并将最符合需求的文档放在首位。
如果有许多满足用户原始需求的文件，应首先交付整合这些文件的报告，最终提交的文件数量应尽可能控制在3个以内。`,
					Type: schema.Object,
					SubParams: map[string]*schema.ParameterInfo{
						"path": {
							Desc: "absolute path",
							Type: schema.String,
						},
						"desc": {
							Desc: "file content description",
							Type: schema.String,
						},
					},
				},
			},
		}),
	}

	SubmitResultReturnDirectly = map[string]bool{
		"SubmitResult": true,
	}
)

func NewToolSubmitResult(op commandline.Operator) tool.InvokableTool {
	return &submitResultTool{op: op}
}

type submitResultTool struct {
	op commandline.Operator
}

func (t *submitResultTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return submitResultToolInfo, nil
}

func (t *submitResultTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	args := &generic.SubmitResult{}
	if err := sonic.Unmarshal([]byte(argumentsInJSON), args); err != nil {
		return "", err
	}

	plan, _ := utils.GetSessionValue[*generic.Plan](ctx, planexecute.PlanSessionKey)
	steps, _ := utils.GetSessionValue[[]planexecute.ExecutedStep](ctx, planexecute.ExecutedStepsSessionKey)

	var fullPlan []*generic.FullPlan
	for i, step := range steps {
		fullPlan = append(fullPlan, &generic.FullPlan{
			TaskID: i + 1,
			Status: generic.PlanStatusDone,
			Desc:   step.Step,
			ExecResult: &generic.SubmitResult{
				IsSuccess: utils.PtrOf(true),
				Result:    step.Result,
			},
		})
	}

	for i := len(steps); i < len(plan.Steps); i++ {
		step := plan.Steps[i]
		fullPlan = append(fullPlan, &generic.FullPlan{
			TaskID: len(fullPlan) + 1,
			Status: generic.PlanStatusSkipped,
			Desc:   step.Desc,
		})
	}

	wd, ok := params.GetTypedContextParams[string](ctx, params.WorkDirSessionKey)
	if !ok {
		return "", fmt.Errorf("work dir not found")
	}

	_ = t.op.WriteFile(ctx, filepath.Join(wd, "final_report.json"), argumentsInJSON)
	_ = generic.Write2PlanMD(ctx, t.op, wd, fullPlan)
	return utils.ToJSONString(&generic.FullPlan{AgentName: compose.END}), nil
}
