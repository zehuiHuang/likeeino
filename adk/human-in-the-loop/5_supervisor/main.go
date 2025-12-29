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
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"

	"likeeino/adk/common/prints"
	"likeeino/adk/common/store"
	"likeeino/adk/common/tool"
)

func main() {
	ctx := context.Background()
	//创建 Supervisor 模式的agent
	//链路
	//traceCloseFn, startSpanFn := trace.AppendCozeLoopCallbackIfConfigured(ctx)
	//defer traceCloseFn(ctx)
	sv, err := buildFinancialSupervisor(ctx)
	if err != nil {
		log.Fatalf("build financial supervisor failed: %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           sv,
		//检查点,作用是中断后,等待用户判定后续操作
		CheckPointStore: store.NewInMemoryStore(),
	})

	query := "查看我的支票账户余额，然后将500美元从支票账户转入储蓄账户。"
	fmt.Println("\n========================================")
	fmt.Println("User Query:", query)
	fmt.Println("========================================")
	fmt.Println()
	//在实际的业务场景中,可以动态生成,若需用户阻断,则需返回生产的CheckPointID,用户判定后将CheckPointID传入
	//ctx, endSpanFn := startSpanFn(ctx, "5_supervisor", query)
	iter := runner.Query(ctx, query, adk.WithCheckPointID("supervisor-1"))

	for {
		lastEvent, interrupted := processEvents(iter)
		if !interrupted {
			break
		}

		interruptCtx := lastEvent.Action.Interrupted.InterruptContexts[0]
		interruptID := interruptCtx.ID

		fmt.Println("\n========================================")
		fmt.Println("APPROVAL REQUIRED")
		fmt.Println("========================================")

		var apResult *tool.ApprovalResult
		for {
			scanner := bufio.NewScanner(os.Stdin)
			fmt.Print("Approve this transaction? (Y/N): ")
			scanner.Scan()
			fmt.Println()
			nInput := scanner.Text()
			if strings.ToUpper(nInput) == "Y" {
				apResult = &tool.ApprovalResult{Approved: true}
				break
			} else if strings.ToUpper(nInput) == "N" {
				fmt.Print("Please provide a reason for denial: ")
				scanner.Scan()
				reason := scanner.Text()
				fmt.Println()
				apResult = &tool.ApprovalResult{Approved: false, DisapproveReason: &reason}
				break
			}
			fmt.Println("Invalid input, please enter Y or N")
		}

		fmt.Println("\n========================================")
		fmt.Println("Resuming execution...")
		fmt.Println("========================================")
		fmt.Println()

		iter, err = runner.ResumeWithParams(ctx, "supervisor-1", &adk.ResumeParams{
			Targets: map[string]any{
				interruptID: apResult,
			},
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("Execution completed")
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
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}
}
