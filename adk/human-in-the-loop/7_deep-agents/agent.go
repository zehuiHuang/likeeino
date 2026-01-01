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
	"os"
	"strconv"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/components/tool/middlewares/errorremover"
	commonModel "likeeino/adk/common/model"
	tool2 "likeeino/adk/common/tool"
)

type rateLimitedModel struct {
	m     model.ToolCallingChatModel
	delay time.Duration
}

func (r *rateLimitedModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	newM, err := r.m.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &rateLimitedModel{newM, r.delay}, nil
}

func (r *rateLimitedModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	time.Sleep(r.delay)
	return r.m.Generate(ctx, input, opts...)
}

func (r *rateLimitedModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	time.Sleep(r.delay)
	return r.m.Stream(ctx, input, opts...)
}

func getRateLimitDelay() time.Duration {
	delayMs := os.Getenv("RATE_LIMIT_DELAY_MS")
	if delayMs == "" {
		return 0
	}
	ms, err := strconv.Atoi(delayMs)
	if err != nil {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

func newRateLimitedModel() model.ToolCallingChatModel {
	delay := getRateLimitDelay()
	if delay == 0 {
		return commonModel.NewChatModel()
	}
	return &rateLimitedModel{
		m:     commonModel.NewChatModel(),
		delay: delay,
	}
}

func buildResearchAgent(ctx context.Context, m model.ToolCallingChatModel) (adk.Agent, error) {
	searchTool, err := NewSearchTool(ctx)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ResearchAgent",
		Description: "一个能够搜索信息并收集各种主题数据的研究代理。",
		Instruction: `你是一名专门从事信息收集的研究人员。
		使用搜索工具为给定任务查找相关信息。
		提供全面且准确的结果。`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{searchTool},
			},
		},
		MaxIterations: 10,
	})
}

func buildAnalysisAgent(ctx context.Context, m model.ToolCallingChatModel) (adk.Agent, error) {
	analyzeTool, err := NewAnalyzeTool(ctx)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "AnalysisAgent",
		Description: "一个处理数据并生成见解的分析代理。",
		Instruction: `您是一名专门处理数据和生成见解的分析代理。
使用分析工具处理数据并提供有意义的分析。
清晰简洁地介绍你的发现。`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{analyzeTool},
			},
		},
		MaxIterations: 10,
	})
}

func NewDataAnalysisDeepAgent(ctx context.Context, m model.ToolCallingChatModel) (adk.Agent, error) {
	//搜集数据agent
	researchAgent, err := buildResearchAgent(ctx, m)
	if err != nil {
		return nil, err
	}
	//分析数据agent
	analysisAgent, err := buildAnalysisAgent(ctx, m)
	if err != nil {
		return nil, err
	}

	//主动提出澄清问题,让用户补充
	followUpTool := tool2.GetFollowUpTool()
	//内部封装有todoTool  和 taskTool
	return deep.New(ctx, &deep.Config{
		Name:        "DataAnalysisAgent",
		Description: "一个用于综合数据分析任务的深度代理，该任务可能需要用户进行澄清。",
		Instruction: `您是一个数据分析代理，可以帮助用户分析市场数据并提供见解。

IMPORTANT: 在开始任何分析之前，您必须首先使用FollowUpTool向用户提出澄清问题，以了解：
1.他们感兴趣的具体市场部门或行业（如技术、金融、医疗保健）
2.他们想要分析的时间段（例如，上个季度、年初至今、具体日期）
3.他们需要什么类型的分析（例如，趋势分析、比较、统计分析）
4.他们对投资建议的风险承受能力（例如，保守、适度、激进）
只有在收到用户的答案后，您才能使用ResearchAgent和AnalysisAgent进行分析。

可用工具：
- FollowUpTool：在进行任何分析之前，先使用此工具提出澄清问题
- ResearchAgent：用于搜索市场数据和信息
- AnalysisAgent：用于分析数据并生成见解`,
		ChatModel: m,
		SubAgents: []adk.Agent{researchAgent, analysisAgent},
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               []tool.BaseTool{followUpTool},
				ToolCallMiddlewares: []compose.ToolMiddleware{errorremover.Middleware()}, // Inject the remove_error middleware.
			},
		},
		MaxIteration: 50,
	})
}
