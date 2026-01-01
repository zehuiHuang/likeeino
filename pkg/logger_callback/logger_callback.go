package logger_callback

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"io"
)

type LoggerCallback struct {
	callbacks.HandlerBuilder // 可以用 callbacks.HandlerBuilder 来辅助实现 callback
}

func (cb *LoggerCallback) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	fmt.Println("==================" + info.Name)
	inputStr, _ := json.MarshalIndent(input, "", "  ")
	fmt.Printf("[OnStart] %s\n", string(inputStr))
	return ctx
}

func (cb *LoggerCallback) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	fmt.Println("=========[OnEnd]=========" + info.Name)
	outputStr, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(outputStr))
	return ctx
}

func (cb *LoggerCallback) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	fmt.Println("=========[OnError]=========" + info.Name)
	fmt.Println(err)
	return ctx
}

func (cb *LoggerCallback) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
	output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {

	var graphInfoName = react.GraphName

	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("[OnEndStream] panic err:", err)
			}
		}()

		defer output.Close() // remember to close the stream in defer

		fmt.Println("=========[OnEndStream]=========" + info.Name)
		for {
			frame, err := output.Recv()
			if errors.Is(err, io.EOF) {
				// finish
				break
			}
			if err != nil {
				fmt.Printf("internal error: %s\n", err)
				return
			}

			s, err := json.Marshal(frame)
			if err != nil {
				fmt.Printf("internal error: %s\n", err)
				return
			}

			if info.Name == graphInfoName { // 仅打印 graph 的输出, 否则每个 stream 节点的输出都会打印一遍
				fmt.Printf("%s: %s\n", info.Name, string(s))
			}
		}

	}()
	return ctx
}

func (cb *LoggerCallback) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo,
	input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	defer input.Close()
	return ctx
}
