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
	"encoding/json"
	"fmt"
	"likeeino/adk/multiagent/integration-excel-agent/utils"
	"strings"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

var (
	bashToolInfo = &schema.ToolInfo{
		Name: "bash",
		Desc: `在bash shell中运行命令
* 调用此工具时，不需要对“command”参数的内容进行XML转义。
* 您无法通过此工具访问互联网。
* 您确实可以通过apt和pip访问常见linux和python包的镜像。
* 状态在命令调用和与用户的讨论中是持久的。
* 要检查文件的特定行范围，例如第10-25行，请尝试“sed-n 10,25p/path/To/file”。
* 请避免使用可能产生大量输出的命令。
* 请在后台运行长期命令，例如“sleep 10&”或在后台启动服务器。`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"command": {
				Type:     "string",
				Desc:     "The command to execute",
				Required: true,
			},
		}),
	}
)

func NewBashTool(op commandline.Operator) tool.InvokableTool {
	return &bashTool{op: op}
}

type bashTool struct {
	op commandline.Operator
}

func (b *bashTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return bashToolInfo, nil
}

type shellInput struct {
	Command string `json:"command"`
}

// InvokableRun 工具自定义执行逻辑
func (b *bashTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	input := &shellInput{}
	err := json.Unmarshal([]byte(argumentsInJSON), input)
	//执行脚本命令
	fmt.Println("///////执行脚本命令///////:" + input.Command)
	if err != nil {
		return "", err
	}
	if len(input.Command) == 0 {
		return "command cannot be empty", nil
	}
	o := tool.GetImplSpecificOptions(&options{b.op}, opts...)
	cmd, err := o.op.RunCommand(ctx, []string{input.Command})
	if err != nil {
		if strings.HasPrefix(err.Error(), "internal error") {
			return err.Error(), nil
		}
		return "", err
	}
	return utils.FormatCommandOutput(cmd), nil
}
