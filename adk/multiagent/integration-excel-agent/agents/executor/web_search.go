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

package executor

import (
	"context"
	"likeeino/adk/multiagent/integration-excel-agent/utils"
	"net/http"
	"time"

	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

func newWebSearchAgent(ctx context.Context) (adk.Agent, error) {
	cm, err := utils.NewChatModel(ctx)
	if err != nil {
		return nil, err
	}

	// 创建自定义的 Transport，设置代理
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	// 创建自定义的 HTTP Client
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second, // 设置超时时间
	}

	searchTool, err := duckduckgo.NewTextSearchTool(ctx, &duckduckgo.Config{
		HTTPClient: httpClient,
	})
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "WebSearchAgent",
		Description: "WebSearchAgent利用ReAct模型分析输入信息，并使用web搜索工具完成任务。",
		Model:       cm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{searchTool},
			},
		},
		MaxIterations: 10,
	})
}
