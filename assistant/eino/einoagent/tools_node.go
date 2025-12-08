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

package einoagent

import (
	"context"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/components/tool"
	"likeeino/pkg/tool/einotool"
	"likeeino/pkg/tool/gitclone"
	"likeeino/pkg/tool/open"
	"likeeino/pkg/tool/task"
	"net/http"
	"time"
)

func GetTools(ctx context.Context) ([]tool.BaseTool, error) {
	einoAssistantTool, err := NewEinoAssistantTool(ctx)
	if err != nil {
		return nil, err
	}

	toolTask, err := NewTaskTool(ctx)
	if err != nil {
		return nil, err
	}

	toolOpen, err := NewOpenFileTool(ctx)
	if err != nil {
		return nil, err
	}

	toolGitClone, err := NewGitCloneFile(ctx)
	if err != nil {
		return nil, err
	}

	toolDDGSearch, err := NewDDGSearch(ctx, nil)
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{
		einoAssistantTool,
		toolTask,
		toolOpen,
		toolGitClone,
		toolDDGSearch,
	}, nil
}

func defaultDDGSearchConfig(ctx context.Context) (*duckduckgo.Config, error) {
	config := &duckduckgo.Config{}

	// 创建自定义的 Transport，设置代理
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	// 创建自定义的 HTTP Client
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second, // 设置超时时间
	}

	// 将自定义的 HTTP Client 赋值给 Config
	config.HTTPClient = httpClient
	return config, nil
}

func NewDDGSearch(ctx context.Context, config *duckduckgo.Config) (tn tool.BaseTool, err error) {
	if config == nil {
		config, err = defaultDDGSearchConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	tn, err = duckduckgo.NewTextSearchTool(ctx, config)
	if err != nil {
		return nil, err
	}
	return tn, nil
}

func NewOpenFileTool(ctx context.Context) (tn tool.BaseTool, err error) {
	return open.NewOpenFileTool(ctx, nil)
}

func NewGitCloneFile(ctx context.Context) (tn tool.BaseTool, err error) {
	return gitclone.NewGitCloneFile(ctx, nil)
}

func NewEinoAssistantTool(ctx context.Context) (tn tool.BaseTool, err error) {
	return einotool.NewEinoAssistantTool(ctx, nil)
}

func NewTaskTool(ctx context.Context) (tn tool.BaseTool, err error) {
	return task.NewTaskTool(ctx, nil)
}
