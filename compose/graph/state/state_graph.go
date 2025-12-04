package main

import (
	"context"
	"errors"
	"github.com/joho/godotenv"
	"io"
	"likeeino/internal/logs"
	"log"
	"os"
	"runtime/debug"
	"strings"
	"unicode/utf8"

	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/coze-dev/cozeloop-go"
)

func main() {
	//Trace
	cozeloopApiToken := os.Getenv("COZELOOP_API_TOKEN")
	cozeloopWorkspaceID := os.Getenv("COZELOOP_WORKSPACE_ID") // use cozeloop trace, from https://loop.coze.cn/open/docs/cozeloop/go-sdk#4a8c980e

	ctx := context.Background()
	var handlers []callbacks.Handler
	if cozeloopApiToken != "" && cozeloopWorkspaceID != "" {
		client, err := cozeloop.NewClient(
			cozeloop.WithAPIToken(cozeloopApiToken),
			cozeloop.WithWorkspaceID(cozeloopWorkspaceID),
		)
		if err != nil {
			panic(err)
		}
		defer client.Close(ctx)
		handlers = append(handlers, clc.NewLoopHandler(client))
	}
	callbacks.AppendGlobalHandlers(handlers...)

	const (
		nodeOfL1 = "invokable"
		nodeOfL2 = "streamable"
		nodeOfL3 = "transformable"
	)
	//定义全局状态
	type testState struct {
		ms []string
	}
	//为WithGenLocalState 定义函数
	gen := func(ctx context.Context) *testState {
		return &testState{}
	}
	//创建graph,并使用WithGenLocalState参数, 使其在不通节点之间共享状态机
	sg := compose.NewGraph[string, string](compose.WithGenLocalState(gen))
	//创建自定义逻辑节点
	l1 := compose.InvokableLambda(func(ctx context.Context, in string) (out string, err error) {
		return "InvokableLambda: " + in, nil
	})
	//定义函数:挂在到某个节点上,接受到入参数据,并暴露出全局状态机,可对其变更
	l1StateToInput := func(ctx context.Context, in string, state *testState) (string, error) {
		state.ms = append(state.ms, in)
		return in, nil
	}
	//定义函数:挂在到某个节点上,接受到出参数据,并暴露出全局状态机,可对其变更
	l1StateToOutput := func(ctx context.Context, out string, state *testState) (string, error) {
		state.ms = append(state.ms, out)
		return out, nil
	}
	//将创建自定义逻辑节点加入的graph,并且将入和出的函数挂在到该自定义的节点上,在执行到该节点时,入和出函数会被执行,对全局状态机进行设置
	_ = sg.AddLambdaNode(nodeOfL1, l1,
		compose.WithStatePreHandler(l1StateToInput), compose.WithStatePostHandler(l1StateToOutput))
	//自定义流式数据生成,将数据按照空格分割,并传入到下一个节点,同时定义了输出类型为*schema.StreamReader[string]
	l2 := compose.StreamableLambda(func(ctx context.Context, input string) (output *schema.StreamReader[string], err error) {
		outStr := "StreamableLambda: " + input

		sr, sw := schema.Pipe[string](utf8.RuneCountInString(outStr))

		go func() {
			defer sw.Close()
			for _, field := range strings.Fields(outStr) {
				sw.Send(field+" ", nil)
			}
		}()

		return sr, nil
	})
	//状态机函数设置
	l2StateToOutput := func(ctx context.Context, out string, state *testState) (string, error) {
		state.ms = append(state.ms, out)
		return out, nil
	}
	//将自定义流式数据处理添加到graph中,并带上状态机函数
	_ = sg.AddLambdaNode(nodeOfL2, l2, compose.WithStatePostHandler(l2StateToOutput))
	//自定义流式数据转换,将上游的流式数据进行转化
	l3 := compose.TransformableLambda(func(ctx context.Context, input *schema.StreamReader[string]) (
		output *schema.StreamReader[string], err error) {

		prefix := "TransformableLambda: "
		sr, sw := schema.Pipe[string](20)

		go func() {

			defer func() {
				if err := recover(); err != nil {
					logs.Errorf("panic occurs: %v\nStack Trace:\n%s", err, string(debug.Stack()))
				}
			}()

			for _, field := range strings.Fields(prefix) {
				sw.Send(field+" ", nil)
			}
			//接收上游流式数据
			for {
				chunk, err := input.Recv()
				if err != nil {
					if err == io.EOF {
						break
					}
					// TODO: how to trace this kind of error in the goroutine of processing sw
					sw.Send(chunk, err)
					break
				}
				//接收后在发送到下游数据
				sw.Send(chunk, nil)

			}
			sw.Close()
		}()

		return sr, nil
	})
	//状体机函数,
	l3StateToOutput := func(ctx context.Context, out string, state *testState) (string, error) {
		state.ms = append(state.ms, out)
		logs.Infof("state result: ")
		for idx, m := range state.ms {
			logs.Infof("    %vth: %v", idx, m)
		}
		return out, nil
	}
	//添加自定义逻辑节点3
	_ = sg.AddLambdaNode(nodeOfL3, l3, compose.WithStatePostHandler(l3StateToOutput))
	//画图: compose.START-》nodeOfL1 -》nodeOfL2 -》nodeOfL3 -》 compose.END
	_ = sg.AddEdge(compose.START, nodeOfL1)

	_ = sg.AddEdge(nodeOfL1, nodeOfL2)

	_ = sg.AddEdge(nodeOfL2, nodeOfL3)

	_ = sg.AddEdge(nodeOfL3, compose.END)
	//编译
	run, err := sg.Compile(ctx)
	if err != nil {
		logs.Errorf("sg.Compile failed, err=%v", err)
		return
	}
	//执行
	out, err := run.Invoke(ctx, "how are you")
	if err != nil {
		logs.Errorf("run.Invoke failed, err=%v", err)
		return
	}
	logs.Infof("invoke result: %v", out)
	//流式执行
	stream, err := run.Stream(ctx, "how are you")
	if err != nil {
		logs.Errorf("run.Stream failed, err=%v", err)
		return
	}
	//流式接收
	for {

		chunk, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			logs.Infof("stream.Recv() failed, err=%v", err)
			break
		}

		logs.Tokenf("%v", chunk)
	}
	stream.Close()

	sr, sw := schema.Pipe[string](1)
	sw.Send("how are you", nil)
	sw.Close()

	stream, err = run.Transform(ctx, sr)
	if err != nil {
		logs.Infof("run.Transform failed, err=%v", err)
		return
	}

	for {

		chunk, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			logs.Infof("stream.Recv() failed, err=%v", err)
			break
		}

		logs.Infof("%v", chunk)
	}
	stream.Close()
}

func init() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}
}
