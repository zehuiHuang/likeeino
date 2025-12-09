package main

import (
	"context"
	"fmt"
	"likeeino/adk/common/model"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"

	tool2 "likeeino/adk/common/tool"
)

// 人机协同

func NewTicketBookingAgent() adk.Agent {
	ctx := context.Background()

	type bookInput struct {
		Location             string `json:"location"`
		PassengerName        string `json:"passenger_name"`
		PassengerPhoneNumber string `json:"passenger_phone_number"`
	}

	getWeather, err := utils.InferTool(
		"BookTicket",
		"this tool can book ticket of the specific location",
		func(ctx context.Context, input bookInput) (output string, err error) {
			return "success", nil
		})
	if err != nil {
		log.Fatal(err)
	}

	a, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "TicketBooker",
		Description: "An agent that can book tickets",
		Instruction: `You are an expert ticket booker.
Based on the user's request, use the "BookTicket" tool to book tickets.`,
		Model: model.NewChatModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					&tool2.InvokableApprovableTool{InvokableTool: getWeather},
				},
			},
		},
	})
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create chatmodel: %w", err))
	}

	return a
}
