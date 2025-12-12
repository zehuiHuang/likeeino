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
	"github.com/joho/godotenv"
	"likeeino/adk/common/prints"
	"likeeino/adk/common/trace"
	"log"
	"time"

	"github.com/cloudwego/eino/adk"
)

/*
*
就是一个简单的supervisor类型的agent,包括一个主代理人,一个搜索信息代理人和一个计算代理人
*/
func main() {
	ctx := context.Background()
	//trace\metric
	traceCloseFn, startSpanFn := trace.AppendCozeLoopCallbackIfConfigured(ctx)
	defer traceCloseFn(ctx)
	//创建Supervisor类型的agent,包括主代理人+搜索信息代理人
	sv, err := buildSupervisor(ctx)
	if err != nil {
		log.Fatalf("build layered supervisor failed: %v", err)
	}

	query := "计算2024年美国和纽约州的国内生产总值。纽约州占美国GDP的百分比是多少？,然后将该百分比乘以1.589。"

	ctx, endSpanFn := startSpanFn(ctx, "layered-supervisor", query)
	iter := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           sv,
	}).Query(ctx, query)

	fmt.Println("\nuser query: ", query)

	var lastMessage adk.Message
	for {
		event, hasEvent := iter.Next()
		if !hasEvent {
			break
		}

		prints.Event(event)

		if event.Output != nil {
			lastMessage, _, err = adk.GetMessage(event)
		}
	}

	endSpanFn(ctx, lastMessage)

	// wait for all span to be ended
	time.Sleep(5 * time.Second)
}
func init() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}
}
