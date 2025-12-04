package main

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"io"
	"log"
	"os"

	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/coze-dev/cozeloop-go"
)

/*
*
关于graph的简单案例
*/
const (
	nodeOfModel  = "model"
	nodeOfPrompt = "prompt"
)

func main() {
	cozeloopApiToken := os.Getenv("COZELOOP_API_TOKEN")
	cozeloopWorkspaceID := os.Getenv("COZELOOP_WORKSPACE_ID") // use cozeloop trace, from https://loop.coze.cn/open/docs/cozeloop/go-sdk#4a8c980e

	ctx := context.Background()
	var handlers []callbacks.Handler
	//https://loop.coze.cn/console/enterprise/personal/open/oauth/apps
	//利用扣子平台,记录观测性的Trace
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

	//创建Graph,入参类型为map[string]any,出参类型为*schema.Message,
	//同时NewGraph函数还可以通过函数进行自定义处理,比如WithGenLocalState函数,维护全局状态
	g := compose.NewGraph[map[string]any, *schema.Message]()

	pt := prompt.FromMessages(
		schema.FString,
		schema.UserMessage("what's the weather in {location}?"),
	)
	//提示词节点
	_ = g.AddChatTemplateNode(nodeOfPrompt, pt) // error check can be skipped here because in Compile, we will check the error happened here too
	//大模型节点,这里通过mockChatModel实现了调用大模型的接口进行mock,并没有真正调用大模型
	_ = g.AddChatModelNode(nodeOfModel, &mockChatModel{}, compose.WithNodeName("ChatModel"))
	//开始画图,以nodeOfPrompt为开头,也就是填写好的提示词节点为开头
	_ = g.AddEdge(compose.START, nodeOfPrompt)
	//将提示词节点后面连接到大模型节点
	_ = g.AddEdge(nodeOfPrompt, nodeOfModel)
	//以大模型节点为结束节点
	_ = g.AddEdge(nodeOfModel, compose.END)
	//设置运行的最大步数,防止无限循环
	r, err := g.Compile(ctx, compose.WithMaxRunSteps(10))
	if err != nil {
		panic(err)
	}
	//入参
	in := map[string]any{"location": "beijing"}
	ret, err := r.Invoke(ctx, in)
	if err != nil {
		panic(err)
	}
	fmt.Println("invoke result: ", ret)

	// stream
	s, err := r.Stream(ctx, in)
	if err != nil {
		panic(err)
	}

	defer s.Close()
	for {
		chunk, err := s.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}

		fmt.Println("stream chunk: ", chunk)
	}
}

type mockChatModel struct{}

func (m *mockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return schema.AssistantMessage("the weather is good", nil), nil
}

func (m *mockChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	sr, sw := schema.Pipe[*schema.Message](0)
	go func() {
		defer sw.Close()
		sw.Send(schema.AssistantMessage("the weather is", nil), nil)
		sw.Send(schema.AssistantMessage(" good", nil), nil)
	}()
	return sr, nil
}

func (m *mockChatModel) BindTools(tools []*schema.ToolInfo) error {
	panic("implement me")
}

func init() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}
}
