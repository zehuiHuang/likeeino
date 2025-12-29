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

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// 编辑文件工具
var (
	editFileToolInfo = &schema.ToolInfo{
		Name: "edit_file",
		Desc: `这是一个用于编辑文件的工具，参数包括文件路径和要编辑的内容。
在任务处理过程中，如果需要创建文件或覆盖文件内容，可以使用此工具。

Notice:
- 如果文件不存在，此工具将使用权限perm创建它（0666）；否则，它将在写入之前截断它，而不更改权限。
- 使用此工具时，请确保文件内容是完整的全文；否则，可能会导致文件内容丢失或错误。
- 仅支持写入文本文件；不支持写入xls/xlsx文件。`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "file absolute path",
				Required: true,
			},
			"content": {
				Type:     schema.String,
				Desc:     "file content",
				Required: true,
			},
		}),
	}
)

func NewEditFileTool(op commandline.Operator) tool.InvokableTool {
	return &editFile{op: op}
}

type editFile struct {
	op commandline.Operator
}

func (e *editFile) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return editFileToolInfo, nil
}

type editFileInput struct {
	Path    string `name:"path"`
	Content string `name:"content"`
}

func (e *editFile) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	input := &editFileInput{}
	err := json.Unmarshal([]byte(argumentsInJSON), input)
	if err != nil {
		return "", err
	}
	fmt.Println("//////edit工具//////:" + input.Path + "和" + input.Content)
	if len(input.Path) == 0 {
		return "path can not be empty", nil
	}
	o := tool.GetImplSpecificOptions(&options{op: e.op}, opts...)
	err = o.op.WriteFile(ctx, input.Path, input.Content)
	if err != nil {
		return err.Error(), nil
	}
	return "edit file success", nil
}
