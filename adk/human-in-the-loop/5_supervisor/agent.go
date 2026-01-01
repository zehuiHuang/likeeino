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
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	commonModel "likeeino/adk/common/model"
	tool2 "likeeino/adk/common/tool"
)

type rateLimitedModel struct {
	m     model.ToolCallingChatModel
	delay time.Duration
}

func (r *rateLimitedModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	newM, err := r.m.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &rateLimitedModel{newM, r.delay}, nil
}

func (r *rateLimitedModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	time.Sleep(r.delay)
	return r.m.Generate(ctx, input, opts...)
}

func (r *rateLimitedModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	time.Sleep(r.delay)
	return r.m.Stream(ctx, input, opts...)
}

func getRateLimitDelay() time.Duration {
	delayMs := os.Getenv("RATE_LIMIT_DELAY_MS")
	if delayMs == "" {
		return 0
	}
	ms, err := strconv.Atoi(delayMs)
	if err != nil {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

func newRateLimitedModel() model.ToolCallingChatModel {
	delay := getRateLimitDelay()
	if delay == 0 {
		//使用ark模型,deepseek模型进过验证可能会有问题
		return commonModel.NewChatModel("ark")
	}
	return &rateLimitedModel{
		m:     commonModel.NewChatModel("ark"),
		delay: delay,
	}
}

func buildAccountAgent(ctx context.Context) (adk.Agent, error) {
	m := newRateLimitedModel()

	type balanceReq struct {
		AccountID string `json:"account_id" jsonschema_description:"The account ID to check balance for"`
	}

	type balanceResp struct {
		AccountID string  `json:"account_id"`
		Balance   float64 `json:"balance"`
		Currency  string  `json:"currency"`
	}

	checkBalance := func(ctx context.Context, req *balanceReq) (*balanceResp, error) {
		//mock账号余额
		balances := map[string]float64{
			"支票账户": 5000.00,
			"储蓄账户": 15000.00,
			"主账户":  5000.00,
		}
		//balances := map[string]float64{
		//	"checking": 5000.00,
		//	"savings":  15000.00,
		//	"main":     5000.00,
		//}
		balance, ok := balances[req.AccountID]
		fmt.Println("//////////////////账户类型ID:", req.AccountID)
		if !ok {
			balance = 1000.00
		}
		return &balanceResp{
			AccountID: req.AccountID,
			Balance:   balance,
			Currency:  "USD",
		}, nil
	}

	balanceTool, err := utils.InferTool("check_balance", "查看特定账户的余额", checkBalance)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "account_agent",
		Description: "负责检查账户信息和余额的代理人",
		Instruction: `您是帐户信息代理.

INSTRUCTIONS:
	- 仅协助处理与账户相关的查询，如检查余额
	- 使用check_balance工具获取帐户信息
	- 完成任务后，直接回复主管
	- 只回复你的工作结果，不要包含任何其他文本。`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{balanceTool},
			},
		},
	})
}

func buildTransactionAgent(ctx context.Context) (adk.Agent, error) {
	//包装了下,使其支持延迟执行
	m := newRateLimitedModel()

	type transferReq struct {
		FromAccount string  `json:"from_account" jsonschema_description:"Source account ID"`
		ToAccount   string  `json:"to_account" jsonschema_description:"Destination account ID"`
		Amount      float64 `json:"amount" jsonschema_description:"Amount to transfer"`
		Currency    string  `json:"currency" jsonschema_description:"Currency code (e.g., USD)"`
	}

	type transferResp struct {
		TransactionID string  `json:"transaction_id"`
		Status        string  `json:"status"`
		FromAccount   string  `json:"from_account"`
		ToAccount     string  `json:"to_account"`
		Amount        float64 `json:"amount"`
		Currency      string  `json:"currency"`
		Message       string  `json:"message"`
	}
	//构造工具的方法(指定的方法签名)
	transfer := func(ctx context.Context, req *transferReq) (*transferResp, error) {
		//mock 转账,在实际业务中可调用api接口并返回数据
		return &transferResp{
			TransactionID: "TXN-2025-001234",
			Status:        "completed",
			FromAccount:   req.FromAccount,
			ToAccount:     req.ToAccount,
			Amount:        req.Amount,
			Currency:      req.Currency,
			Message:       fmt.Sprintf("Successfully transferred %.2f %s from %s to %s", req.Amount, req.Currency, req.FromAccount, req.ToAccount),
		}, nil
	}
	//创建工具的方法,只需要提供指定的方法签名即可(可以指定入参]出参的结构体,以及覆盖执行逻辑)
	transferTool, err := utils.InferTool("transfer_funds", "在账户之间转账。这是一项需要用户批准的敏感操作。", transfer)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "transaction_agent",
		Description: "负责执行资金转账等金融交易的代理人",
		Instruction: `您是交易处理代理。

INSTRUCTIONS:
	- 仅协助处理与交易相关的任务，如资金转账
	- 使用transfer_funds工具执行转账
	- transfer_funds工具在执行前需要用户批准
	- 完成任务后，直接回复主管
	- 只回复你的工作结果，不要包含任何其他文本。`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					&tool2.InvokableApprovableTool{InvokableTool: transferTool},
				},
			},
		},
	})
}

func buildFinancialSupervisor(ctx context.Context) (adk.Agent, error) {
	m := newRateLimitedModel()

	sv, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "financial_supervisor",
		Description: "负责协调财务任务的主管代理人",
		Instruction: `您是一名财务顾问主管，负责管理两名代理人：

- an account_agent: 将与帐户相关的任务分配给此代理（检查余额、帐户信息）
- a transaction_agent: 将交易相关任务分配给此代理（资金转账、付款）

INSTRUCTIONS:
	- 分析用户的请求并委派给相应的代理
	- 对于涉及检查余额和转账的请求，首先委托给account_agent，然后委托给transaction_agent
	- 一次将工作分配给一个代理，不要并行呼叫代理
	- 不要自己做任何工作——一定要委托给合适的代理人
	- 所有任务完成后，为用户总结结果`,
		Model: m,
		Exit:  &adk.ExitTool{},
	})
	if err != nil {
		return nil, err
	}
	//账号查询等操作代理
	accountAgent, err := buildAccountAgent(ctx)
	if err != nil {
		return nil, err
	}
	//账户交易等操作代理
	transactionAgent, err := buildTransactionAgent(ctx)
	if err != nil {
		return nil, err
	}

	return supervisor.New(ctx, &supervisor.Config{
		Supervisor: sv,
		SubAgents:  []adk.Agent{accountAgent, transactionAgent},
	})
}
