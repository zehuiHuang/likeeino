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
	"bufio"
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"likeeino/adk/common/model"
	"likeeino/adk/common/prints"
	"likeeino/adk/multiagent/integration-project-manager/agents"
	"likeeino/pkg/retriever"
	"log"
	"os"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

func main() {
	ctx := context.Background()

	tcm := model.NewChatModel()
	// 代理的初始化聊天模型
	//tcm, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
	//	APIKey: os.Getenv("OPENAI_API_KEY"),
	//	Model:  os.Getenv("OPENAI_MODEL"),
	//	//BaseURL: os.Getenv("OPENAI_BASE_URL"),
	//	//ByAzure: func() bool {
	//	//	return os.Getenv("OPENAI_BY_AZURE") == "true"
	//	//}(),
	//})
	//if err != nil {
	//	log.Fatal(err)
	//}

	// Init research agent
	//子agent,支持中断和web search
	researchAgent, err := agents.NewResearchAgent(ctx, tcm)
	if err != nil {
		log.Fatal(err)
	}
	//初始化检索器
	rtr, err := retriever.NewRetriever(ctx)

	// Init code agent
	//子agent,可以利用知识库 来生产高质量的代码
	codeAgent, err := agents.NewCodeAgent(ctx, tcm, rtr)
	if err != nil {
		log.Fatal(err)
	}

	// Init technical agent
	//子agent,帮助代码的review
	reviewAgent, err := agents.NewReviewAgent(ctx, tcm)
	if err != nil {
		log.Fatal(err)
	}

	// Init project manager agent
	//主agent
	s, err := agents.NewProjectManagerAgent(ctx, tcm)
	if err != nil {
		log.Fatal(err)
	}

	// Combine agents into ADK supervisor pattern
	// Supervisor: project manager
	// Sub-agents: researcher / coder / reviewer
	supervisorAgent, err := supervisor.New(ctx, &supervisor.Config{
		Supervisor: s,
		SubAgents:  []adk.Agent{researchAgent, codeAgent, reviewAgent},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Init Agent runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           supervisorAgent,
		EnableStreaming: true,
		CheckPointStore: newInMemoryStore(),
	})

	// Replace it with your own query
	// When using the following query, researchAgent will interrupt and prompt the user to input the specific research subject via stdin.
	//query := "please give me a report about advantages of  xxx" //替换成自己的问题
	query := "使用  eino 框架给我生成一份人机协同的demo案例 " //替换成自己的问题
	checkpointID := "1"

	// The researchAgent may require users to input information multiple times
	// Therefore, the following flags, "interrupted" and "finished," are used to support multiple interruptions and resumptions.
	interrupted := false
	finished := false

	for !finished {
		var iter *adk.AsyncIterator[*adk.AgentEvent]

		if !interrupted {
			iter = runner.Query(ctx, query, adk.WithCheckPointID(checkpointID))
		} else {
			scanner := bufio.NewScanner(os.Stdin)
			fmt.Print("\ninput additional context for web search: ")
			scanner.Scan()
			fmt.Println()
			nInput := scanner.Text()

			iter, err = runner.Resume(ctx, checkpointID, adk.WithToolOptions([]tool.Option{agents.WithNewInput(nInput)}))
			if err != nil {
				log.Fatal(err)
			}
		}

		interrupted = false

		for {
			event, ok := iter.Next()
			if !ok {
				if !interrupted {
					finished = true
				}
				break
			}
			if event.Err != nil {
				log.Fatal(event.Err)
			}
			if event.Action != nil {
				if event.Action.Interrupted != nil {
					interrupted = true
				}
				if event.Action.Exit {
					finished = true
				}
			}
			prints.Event(event)
		}
	}
}

func newInMemoryStore() compose.CheckPointStore {
	return &inMemoryStore{
		mem: map[string][]byte{},
	}
}

type inMemoryStore struct {
	mem map[string][]byte
}

func (i *inMemoryStore) Set(ctx context.Context, key string, value []byte) error {
	i.mem[key] = value
	return nil
}

func (i *inMemoryStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	v, ok := i.mem[key]
	return v, ok, nil
}

func init() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}
}
