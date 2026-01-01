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
	"encoding/gob"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"

	"likeeino/adk/common/prints"
	"likeeino/adk/common/store"
	"likeeino/adk/common/tool"
)

func main() {
	ctx := context.Background()

	agent, err := NewTravelPlanningAgent(ctx)
	if err != nil {
		log.Fatalf("failed to create travel planning agent: %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
		CheckPointStore: store.NewInMemoryStore(),
	})

	//query := `Plan a 3-day trip to Tokyo starting from New York on 2025-10-15.
	//I need to book flights and a hotel. Also recommend some must-see attractions.
	//Today is 2025-09-01.`

	query := `计划2025年10月15日从New York开始为期3天的Tokyo之旅。我自己需要预订航班和酒店,档次不限要求不限。还推荐一些必看景点。今天是2025-09-01。`

	fmt.Println("\n========================================")
	fmt.Println("User Query:", query)
	fmt.Println("========================================")
	fmt.Println()

	iter := runner.Query(ctx, query, adk.WithCheckPointID("travel-plan-1"))

	for {
		lastEvent, interrupted := processEvents(iter)
		if !interrupted {
			break
		}

		interruptCtx := lastEvent.Action.Interrupted.InterruptContexts[0]
		interruptID := interruptCtx.ID
		//由出发中断的tool提供的信息
		reInfo := interruptCtx.Info.(*tool.ReviewEditInfo)

		fmt.Println("\n========================================")
		fmt.Println("REVIEW REQUIRED")
		fmt.Println("========================================")
		fmt.Printf("Tool: %s\n", reInfo.ToolName)
		fmt.Printf("Arguments: %s\n", reInfo.ArgumentsInJSON)
		fmt.Println("----------------------------------------")
		fmt.Println("Options:")
		fmt.Println("  - Type 'ok' to approve as-is")
		fmt.Println("  - Type 'n' to reject")
		fmt.Println("  - Or enter modified JSON arguments")
		fmt.Println("----------------------------------------")

		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("Your choice: ")
		scanner.Scan()
		nInput := scanner.Text()
		fmt.Println()

		result := &tool.ReviewEditResult{}
		switch strings.ToLower(strings.TrimSpace(nInput)) {
		case "ok", "yes", "y":
			result.NoNeedToEdit = true
		case "n", "no":
			result.Disapproved = true
			fmt.Print("Reason for rejection (optional): ")
			scanner.Scan()
			reason := scanner.Text()
			if reason != "" {
				result.DisapproveReason = &reason
			}
		default:
			result.EditedArgumentsInJSON = &nInput
		}

		reInfo.ReviewResult = result

		fmt.Println("\n========================================")
		fmt.Println("Resuming execution...")
		fmt.Println("========================================")
		fmt.Println()

		iter, err = runner.ResumeWithParams(ctx, "travel-plan-1", &adk.ResumeParams{
			Targets: map[string]any{
				interruptID: reInfo,
			},
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("Travel planning completed!")
	fmt.Println("========================================")
}

func processEvents(iter *adk.AsyncIterator[*adk.AgentEvent]) (*adk.AgentEvent, bool) {
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
		return nil, false
	}
	if lastEvent.Action != nil && lastEvent.Action.Interrupted != nil {
		return lastEvent, true
	}
	return lastEvent, false
}
func init() {
	// 注册 gob 类型，以便能够序列化/反序列化 ExecutedStep 类型
	gob.Register([]planexecute.ExecutedStep{})
	gob.Register(planexecute.ExecutedStep{})

	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}
}
