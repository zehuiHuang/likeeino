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

package main

import (
	"context"
	"fmt"
	"likeeino/adk/common/model"
	l2 "likeeino/pkg/tool/flow"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

func buildSearchAgent(ctx context.Context) (adk.Agent, error) {
	m := model.NewChatModel()

	type searchReq struct {
		Query string `json:"query"`
	}

	type searchResp struct {
		Result string `json:"result"`
	}

	search := func(ctx context.Context, req *searchReq) (*searchResp, error) {
		//此处为mock数据,可以替换为真正的搜索
		return &searchResp{
			Result: "2024年，美国GDP为29.18万亿美元，纽约州GDP为2.297万亿美元",
		}, nil
	}

	searchTool, err := l2.SafeInferTool("search", "在互联网上搜索信息", search)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "research_agent",
		Description: "负责在互联网上搜索信息的代理人",
		Instruction: `
		你是一名研究人员.


        指令:
        - 仅协助完成与研究相关的任务，不要做任何数学题
        - 完成任务后，直接回复主管
        - 只回复你的工作结果，不要包含任何其他文本。`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{searchTool},
				UnknownToolsHandler: func(ctx context.Context, name, input string) (string, error) {
					return fmt.Sprintf("unknown tool: %s", name), nil
				},
			},
		},
	})
}

func buildSubtractAgent(ctx context.Context) (adk.Agent, error) {
	m := model.NewChatModel()

	type subtractReq struct {
		A float64 `json:"a"`
		B float64 `json:"b"`
	}

	type subtractResp struct {
		Result float64
	}

	subtract := func(ctx context.Context, req *subtractReq) (*subtractResp, error) {
		return &subtractResp{
			Result: req.A - req.B,
		}, nil
	}

	subtractTool, err := l2.SafeInferTool("subtract", "subtract two numbers", subtract)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "subtract_agent",
		Description: "负责进行数学减法的代理",
		Instruction: `
		你是个数学减法师


        指令:
        - 仅协助数学减法相关任务
        - 完成任务后，直接回复主管
        - 只回复你的工作结果，不要包含任何其他文本。`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{subtractTool},
				UnknownToolsHandler: func(ctx context.Context, name, input string) (string, error) {
					return fmt.Sprintf("unknown tool: %s", name), nil
				},
			},
		},
	})
}

func buildMultiplyAgent(ctx context.Context) (adk.Agent, error) {
	m := model.NewChatModel()

	type multiplyReq struct {
		A float64 `json:"a"`
		B float64 `json:"b"`
	}

	type multiplyResp struct {
		Result float64
	}

	multiply := func(ctx context.Context, req *multiplyReq) (*multiplyResp, error) {
		return &multiplyResp{
			Result: req.A * req.B,
		}, nil
	}

	multiplyTool, err := l2.SafeInferTool("multiply", "multiply two numbers", multiply)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "multiply_agent",
		Description: "the agent responsible to do math multiplications",
		Instruction: `
		You are a math multiplication agent.


        INSTRUCTIONS:
        - Assist ONLY with math multiplication-related tasks
        - After you're done with your tasks, respond to the supervisor directly
        - Respond ONLY with the results of your work, do NOT include ANY other text.`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{multiplyTool},
				UnknownToolsHandler: func(ctx context.Context, name, input string) (string, error) {
					return fmt.Sprintf("unknown tool: %s", name), nil
				},
			},
		},
	})
}

func buildDivideAgent(ctx context.Context) (adk.Agent, error) {
	m := model.NewChatModel()

	type divideReq struct {
		A float64 `json:"a"`
		B float64 `json:"b"`
	}

	type divideResp struct {
		Result float64
	}

	divide := func(ctx context.Context, req *divideReq) (*divideResp, error) {
		return &divideResp{
			Result: req.A / req.B,
		}, nil
	}

	divideTool, err := l2.SafeInferTool("divide", "divide two numbers", divide)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "divide_agent",
		Description: "the agent responsible to do math division",
		Instruction: `
		You are a math division agent.


        INSTRUCTIONS:
        - Assist ONLY with math division-related tasks
        - After you're done with your tasks, respond to the supervisor directly
        - Respond ONLY with the results of your work, do NOT include ANY other text.`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{divideTool},
				UnknownToolsHandler: func(ctx context.Context, name, input string) (string, error) {
					return fmt.Sprintf("unknown tool: %s", name), nil
				},
			},
		},
	})
}

func buildMathAgent(ctx context.Context) (adk.Agent, error) {
	m := model.NewChatModel()

	sa, err := buildSubtractAgent(ctx)
	if err != nil {
		return nil, err
	}

	ma, err := buildMultiplyAgent(ctx)
	if err != nil {
		return nil, err
	}

	da, err := buildDivideAgent(ctx)
	if err != nil {
		return nil, err
	}

	mathA, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "math_agent",
		Description: "the agent responsible to do math",
		Instruction: `
		You are a math agent.


        INSTRUCTIONS:
        - Assist ONLY with math-related tasks
        - After you're done with your tasks, respond to the supervisor directly
        - Respond ONLY with the results of your work, do NOT include ANY other text.
		- You are yourself also a supervisor managing three agents:
		- an subtract_agent, a multiply_agent, a divide_agent. Assign math-related tasks to these agents.
		- Assign work to one agent at a time, do not call agents in parallel.
		- Do not do any real math work yourself, always transfer to your sub agents to do actual computation.`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				UnknownToolsHandler: func(ctx context.Context, name, input string) (string, error) {
					return fmt.Sprintf("unknown tool: %s", name), nil
				},
			},
		},
	})

	return supervisor.New(ctx, &supervisor.Config{
		Supervisor: mathA,
		SubAgents:  []adk.Agent{sa, ma, da},
	})
}

func buildSupervisor(ctx context.Context) (adk.Agent, error) {
	m := model.NewChatModel()

	sv, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "supervisor",
		Description: "负责监督任务的代理人",
		Instruction: `
		您是管理两名代理人的主管:

        - a research agent. Assign research-related tasks to this agent
        - 数学代理人。将数学相关任务分配给此代理
        一次将工作分配给一个代理，不要并行呼叫代理。
        不要自己做任何工作。`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				//用于处理 agent 尝试调用不存在工具的情况，提供错误处理和反馈机制
				UnknownToolsHandler: func(ctx context.Context, name, input string) (string, error) {
					return fmt.Sprintf("unknown tool: %s", name), nil
				},
			},
		},
		//用于让主管 agent 在完成任务后明确表示工作流的结束
		Exit: &adk.ExitTool{},
	})
	if err != nil {
		return nil, err
	}
	//搜索数据
	searchAgent, err := buildSearchAgent(ctx)
	if err != nil {
		return nil, err
	}
	//数学计算
	mathAgent, err := buildMathAgent(ctx)
	if err != nil {
		return nil, err
	}

	return supervisor.New(ctx, &supervisor.Config{
		Supervisor: sv,
		SubAgents:  []adk.Agent{searchAgent, mathAgent},
	})
}
