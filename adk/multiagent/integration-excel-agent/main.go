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
	"github.com/joho/godotenv"
	"likeeino/adk/common/prints"
	"likeeino/adk/common/trace"
	"likeeino/adk/multiagent/integration-excel-agent/agents/executor"
	"likeeino/adk/multiagent/integration-excel-agent/agents/planner"
	"likeeino/adk/multiagent/integration-excel-agent/agents/replanner"
	"likeeino/adk/multiagent/integration-excel-agent/agents/report"
	"likeeino/adk/multiagent/integration-excel-agent/generic"
	"likeeino/adk/multiagent/integration-excel-agent/params"
	"likeeino/adk/multiagent/integration-excel-agent/utils"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

func main() {
	// Set your own query here. e.g.
	// query := schema.UserMessage("统计附件文件中推荐的小说名称及推荐次数，并将结果写到文件中。凡是带有《》内容都是小说名称，形成表格，表头为小说名称和推荐次数，同名小说只列一行，推荐次数相加")
	// query := schema.UserMessage("Count the recommended novel names and recommended times in the attachment file, and write the results into the file. The content with "" is the name of the novel, forming a table. The header is the name of the novel and the number of recommendations. The novels with the same name are listed in one row, and the number of recommendations is added")

	// query := schema.UserMessage("读取 模拟出题.csv 中的表格内容，规范格式将题目、答案、解析、选项放在同一行，简答题只把答案写入解析即可")
	// query := schema.UserMessage("Read the table content in the 模拟出题.csv, put the question, answer, resolution and options in the same line in a standardized format, and simply write the answer into the resolution")

	query := schema.UserMessage("请帮我将 questions.csv 表格中的第一列提取到一个新的 csv 中")
	// query := schema.UserMessage("Please help me extract the first column in question.csv table into a new csv")

	ctx := context.Background()
	//链路
	traceCloseFn, startSpanFn := trace.AppendCozeLoopCallbackIfConfigured(ctx)
	defer traceCloseFn(ctx)
	//创建agent
	agent, err := newExcelAgent(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// uuid as task id
	uuid := uuid.New().String()
	//创建runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})
	//获取当前路径
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	//找到输入信息路径
	var inputFileDir, workdir string
	if env := os.Getenv("EXCEL_AGENT_INPUT_DIR"); env != "" {
		inputFileDir = env
	} else {
		inputFileDir = filepath.Join(wd, "adk/multiagent/integration-excel-agent/playground/input")
	}
	//找到输出信息路径
	if env := os.Getenv("EXCEL_AGENT_WORK_DIR"); env != "" {
		workdir = filepath.Join(env, uuid)
	} else {
		workdir = filepath.Join(wd, "adk/multiagent/integration-excel-agent/playground", uuid)
	}

	if err = os.Mkdir(workdir, 0755); err != nil {
		log.Fatal(err)
	}

	if err = os.CopyFS(workdir, os.DirFS(inputFileDir)); err != nil {
		log.Fatal(err)
	}

	previews, err := generic.PreviewPath(workdir)
	if err != nil {
		log.Fatal(err)
	}
	//将输入信息、所在路径等信息添加到上下文中,方面后续的使用
	ctx = params.InitContextParams(ctx)
	params.AppendContextParams(ctx, map[string]interface{}{
		params.FilePathSessionKey:            inputFileDir,                 //输入信息路径
		params.WorkDirSessionKey:             workdir,                      //工作空间
		params.UserAllPreviewFilesSessionKey: utils.ToJSONString(previews), //文件转成对象-》转json
		params.TaskIDKey:                     uuid,
	})

	ctx, endSpanFn := startSpanFn(ctx, "plan-execute-replan", query)
	//执行-----agent---- >
	iter := runner.Run(ctx, []*schema.Message{query})

	var (
		lastMessage       adk.Message
		lastMessageStream *schema.StreamReader[adk.Message]
	)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			if lastMessageStream != nil {
				lastMessageStream.Close()
			}
			if event.Output.MessageOutput.IsStreaming {
				cpStream := event.Output.MessageOutput.MessageStream.Copy(2)
				event.Output.MessageOutput.MessageStream = cpStream[0]
				lastMessage = nil
				lastMessageStream = cpStream[1]
			} else {
				lastMessage = event.Output.MessageOutput.Message
				lastMessageStream = nil
			}
		}
		prints.Event(event)
	}

	if lastMessage != nil {
		endSpanFn(ctx, lastMessage)
	} else if lastMessageStream != nil {
		msg, _ := schema.ConcatMessageStream(lastMessageStream)
		endSpanFn(ctx, msg)
	} else {
		endSpanFn(ctx, "finished without output message")
	}

	time.Sleep(time.Second * 30)
}

// 创建代理
func newExcelAgent(ctx context.Context) (adk.Agent, error) {
	//本地文件操作接口实现(官方还给出docker等环境的操作实现)
	operator := &LocalOperator{}

	//规划代理--
	p, err := planner.NewPlanner(ctx, operator)
	if err != nil {
		return nil, err
	}

	//执行代理--
	e, err := executor.NewExecutor(ctx, operator)
	if err != nil {
		return nil, err
	}

	//重规划代理--
	rp, err := replanner.NewReplanner(ctx, operator)
	if err != nil {
		return nil, err
	}
	//创建规划执行代理,分别为规划、执行、重规划
	planExecuteAgent, err := planexecute.New(ctx, &planexecute.Config{
		Planner:       p,
		Executor:      e,
		Replanner:     rp,
		MaxIterations: 20,
	})
	if err != nil {
		return nil, err
	}

	reportAgent, err := report.NewReportAgent(ctx, operator)
	if err != nil {
		return nil, err
	}
	//创建主代理,子代理分别为planExecuteAgent和reportAgent
	agent, err := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        "SequentialAgent",
		Description: "sequential agent",
		SubAgents: []adk.Agent{
			planExecuteAgent, reportAgent,
		},
	})
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func init() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}
}
