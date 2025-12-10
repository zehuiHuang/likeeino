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
	"likeeino/adk/common/tool"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
)

/*
*
人机交互,认为进行参数调整,覆盖、拒绝或直接通过
*/
func main() {
	ctx := context.Background()
	a := NewTicketAgent()
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           a,
		CheckPointStore: store.NewInMemoryStore(),
	})

	iter := runner.Query(ctx, "book a ticket for Martin, to Beijing, on 2025-12-01, the phone number is 1234567. directly call tool.", adk.WithCheckPointID("1"))

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
			fmt.Println("\n--- Conversation Finished ---")
			return
		}

		if lastEvent.Action == nil || lastEvent.Action.Interrupted == nil {
			fmt.Println("\n--- Conversation Finished ---")
			return
		}

		// Handle the review-and-edit interrupt
		interruptCtx := lastEvent.Action.Interrupted.InterruptContexts[0]
		reInfo := interruptCtx.Info.(*tool.ReviewEditInfo)

		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("\nYour input: ")
		scanner.Scan()
		fmt.Println()
		nInput := scanner.Text()

		result := &tool.ReviewEditResult{}
		switch strings.ToLower(nInput) {
		case "no need to edit":
			result.NoNeedToEdit = true
		case "n":
			result.Disapproved = true
			fmt.Print("Reason for disapproval (optional): ")
			scanner.Scan()
			reason := scanner.Text()
			if reason != "" {
				result.DisapproveReason = &reason
			}
		default:
			result.EditedArgumentsInJSON = &nInput
		}
		reInfo.ReviewResult = result

		var err error
		iter, err = runner.ResumeWithParams(ctx, "1", &adk.ResumeParams{
			Targets: map[string]any{
				interruptCtx.ID: reInfo,
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
