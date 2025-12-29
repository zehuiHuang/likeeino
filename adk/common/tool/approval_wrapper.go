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

package tool

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type ApprovalInfo struct {
	ToolName        string
	ArgumentsInJSON string
	ToolCallID      string
}

type ApprovalResult struct {
	Approved         bool
	DisapproveReason *string
}

func (ai *ApprovalInfo) String() string {
	return fmt.Sprintf("tool '%s' interrupted with arguments '%s', waiting for your approval, "+
		"please answer with Y/N",
		ai.ToolName, ai.ArgumentsInJSON)
}

func init() {
	schema.Register[*ApprovalInfo]()
}

// 工具包装,可以执行中断

type InvokableApprovableTool struct {
	tool.InvokableTool
}

func (i InvokableApprovableTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return i.InvokableTool.Info(ctx)
}

// 自定义工具包装类,效果就是在执行到该工具时,会进行中断,

func (i InvokableApprovableTool) InvokableRun(ctx context.Context, argumentsInJSON string,
	opts ...tool.Option) (string, error) {

	toolInfo, err := i.Info(ctx)
	if err != nil {
		return "", err
	}

	wasInterrupted, _, storedArguments := compose.GetInterruptState[string](ctx)
	//初始调用、中断并等待批准
	if !wasInterrupted { // initial invocation, interrupt and wait for approval
		return "", compose.StatefulInterrupt(ctx, &ApprovalInfo{
			ToolName:        toolInfo.Name,
			ArgumentsInJSON: argumentsInJSON,
			ToolCallID:      compose.GetToolCallID(ctx),
		}, argumentsInJSON)
	}

	isResumeTarget, hasData, data := compose.GetResumeContext[*ApprovalResult](ctx)
	//中断但未明确恢复，再次中断并等待批准
	if !isResumeTarget { // was interrupted but not explicitly resumed, reinterrupt and wait for approval again
		return "", compose.StatefulInterrupt(ctx, &ApprovalInfo{
			ToolName:        toolInfo.Name,
			ArgumentsInJSON: storedArguments,
			ToolCallID:      compose.GetToolCallID(ctx),
		}, storedArguments)
	}
	if !hasData {
		return "", fmt.Errorf("tool '%s' resumed with no data", toolInfo.Name)
	}
	//当中断被恢复时,且用户同意继续执行,执行真正的InvokableRun
	if data.Approved {
		return i.InvokableTool.InvokableRun(ctx, storedArguments, opts...)
	}

	if data.DisapproveReason != nil {
		return fmt.Sprintf("tool '%s' disapproved, reason: %s", toolInfo.Name, *data.DisapproveReason), nil
	}

	return fmt.Sprintf("tool '%s' disapproved", toolInfo.Name), nil
}
