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

package generic

import (
	"encoding/json"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/schema"
)

// Step 和 Plan为规划代理的结构体
type Step struct {
	Index int    `json:"index"`
	Desc  string `json:"desc"`
}

// 实现了 Plan的接口

type Plan struct {
	Steps []Step `json:"steps"`
}

func (p *Plan) FirstStep() string {
	if len(p.Steps) == 0 {
		return ""
	}
	stepStr, _ := sonic.MarshalString(p.Steps[0])
	return stepStr
}

func (p *Plan) MarshalJSON() ([]byte, error) {
	type Alias Plan
	return json.Marshal((*Alias)(p))
}

func (p *Plan) UnmarshalJSON(bytes []byte) error {
	type Alias Plan
	a := (*Alias)(p)
	return json.Unmarshal(bytes, a)
}

var PlanToolInfo = &schema.ToolInfo{
	Name: "create_plan",
	Desc: "生成一个结构化的、循序渐进的执行计划，以解决给定的复杂任务。计划中的每个步骤都必须分配给一个专门的代理，并且必须有一个清晰、可操作的描述。",
	//定义调用该工具时的入参结构
	ParamsOneOf: schema.NewParamsOneOfByParams(
		map[string]*schema.ParameterInfo{
			"steps": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.Object,
					SubParams: map[string]*schema.ParameterInfo{
						"index": {
							Type:     schema.Integer,
							Desc:     "步骤在整体计划中的顺序号（必须从1开始，每个后续步骤递增1)",
							Required: true,
						},
						"desc": {
							Type: schema.String,
							//Desc:     "A clear, concise, and actionable description of the specific task to be performed in this step. It should be a direct instruction for the assigned agent.",
							Desc:     "此步骤中要执行的具体任务的清晰、简洁和可操作的描述。这应该是对指定代理人的直接指示。",
							Required: true,
						},
					},
				},
				//Desc:     "different steps to follow, should be in sorted order",
				Desc:     "要遵循的不同步骤应按顺序排列",
				Required: true,
			},
		},
	),
}
