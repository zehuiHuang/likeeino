package main

import (
	"bufio"
	"context"
	"fmt"
	"likeeino/adk/common/prints"
	"likeeino/adk/common/store"
	"likeeino/adk/common/tool"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"github.com/cloudwego/eino/adk"
)

func main() {
	ctx := context.Background()
	//创建机票agent
	a := NewTicketBookingAgent()
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true, // you can disable streaming here
		Agent:           a,

		// provide a CheckPointStore for eino to persist the execution state of the agent for later resumption.
		// Here we use an in-memory store for simplicity.
		// In the real world, you can use a distributed store like Redis to persist the checkpoints.
		CheckPointStore: store.NewInMemoryStore(),
	})
	iter := runner.Query(ctx, "book a ticket for Martin, to Beijing, on 2025-12-01, the phone number is 1234567. directly call tool.", adk.WithCheckPointID("1"))
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
	if lastEvent.Action == nil || lastEvent.Action.Interrupted == nil {
		log.Fatal("last event is not an interrupt event")
	}

	// this interruptID is crucial 'locator' for Eino to know where the interrupt happens,
	// so when resuming later, you have to provide this same `interruptID` along with the approval result back to Eino
	interruptID := lastEvent.Action.Interrupted.InterruptContexts[0].ID
	for i := range lastEvent.Action.Interrupted.InterruptContexts {
		fmt.Println("InterruptContexts ID:" + lastEvent.Action.Interrupted.InterruptContexts[i].ID)
	}
	var apResult *tool.ApprovalResult
	for {

		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("your input here: ")
		scanner.Scan()
		fmt.Println()
		nInput := scanner.Text()
		if strings.ToUpper(nInput) == "Y" {
			apResult = &tool.ApprovalResult{Approved: true}
			break
		} else if strings.ToUpper(nInput) == "N" {
			// Prompt for reason when denying
			fmt.Print("Please provide a reason for denial: ")
			scanner.Scan()
			reason := scanner.Text()
			fmt.Println()
			apResult = &tool.ApprovalResult{Approved: false, DisapproveReason: &reason}
			break
		}

		fmt.Println("invalid input, please input Y or N")
	}

	// here we directly resumes right in the same instance where the original `Runner.Query` happened.
	// In the real world, the original `Runner.Run/Query` and the subsequent `Runner.ResumeWithParams`
	// can happen in different processes or machines, as long as you use the same `CheckPointID`,
	// and you provided a distributed `CheckPointStore` when creating the `Runner` instance.
	//恢复执行,即传入检查点ID和恢复标识,并将修改的参数apResult传入
	iter, err := runner.ResumeWithParams(ctx, "1", &adk.ResumeParams{
		Targets: map[string]any{
			interruptID: apResult,
		},
	})

	if err != nil {
		log.Fatal(err)
	}
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			log.Fatal(event.Err)
		}

		prints.Event(event)
	}
}

func init() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}
}
