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
	"likeeino/adk/common/prints"
	"likeeino/adk/common/store"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
)

func main() {
	ctx := context.Background()
	//创建循环智能体(包括多个子agent)
	a := NewWriterAgent()
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true, // you can disable streaming here
		Agent:           a,
		CheckPointStore: store.NewInMemoryStore(),
	})
	iter := runner.Query(ctx, "write a short poem about potato, in under 20 words", adk.WithCheckPointID("1"))

	for {
		var lastEvent *adk.AgentEvent
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				log.Fatal(event.Err)
			}

			prints.Event(event)

			lastEvent = event
		}

		if lastEvent == nil {
			log.Fatal("last event is nil")
		}

		if lastEvent.Action != nil && lastEvent.Action.Exit {
			return
		}

		if lastEvent.Action == nil || lastEvent.Action.Interrupted == nil {
			log.Fatal("last event is not an interrupt event")
		}

		reInfo := lastEvent.Action.Interrupted.InterruptContexts[0].Info.(*FeedbackInfo)
		interruptID := lastEvent.Action.Interrupted.InterruptContexts[0].ID

		for {
			scanner := bufio.NewScanner(os.Stdin)
			fmt.Print("your input here: ")
			scanner.Scan()
			fmt.Println()
			nInput := scanner.Text()
			if strings.ToUpper(nInput) == "NO NEED TO EDIT" {
				reInfo.NoNeedToEdit = true
				break
			} else {
				reInfo.Feedback = &nInput
				break
			}
		}

		var err error
		iter, err = runner.ResumeWithParams(ctx, "1", &adk.ResumeParams{
			Targets: map[string]any{
				interruptID: reInfo,
			},
		})
		if err != nil {
			log.Fatal(err)
		}
	}
}

func init() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}
}
