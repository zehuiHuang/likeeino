package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/joho/godotenv"
	"io"
	"likeeino/internal/logs"
	"log"
	"os"
	"time"

	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	callbacks2 "github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino/utils/callbacks"
	"github.com/coze-dev/cozeloop-go"
)

/*
*
主要是让两个chatModel 相互对话, 然后设置分支条件,如果>=3时,直接END
*/
func main() {
	openAIBaseURL := os.Getenv("OPENAI_BASE_URL")
	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	modelName := os.Getenv("OPENAI_MODEL_NAME")

	cozeloopApiToken := os.Getenv("COZELOOP_API_TOKEN")
	cozeloopWorkspaceID := os.Getenv("COZELOOP_WORKSPACE_ID") // use cozeloop trace, from https://loop.coze.cn/open/docs/cozeloop/go-sdk#4a8c980e

	ctx := context.Background()
	var handlers []callbacks2.Handler
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
	callbacks2.AppendGlobalHandlers(handlers...)

	type state struct {
		currentRound int
		msgs         []*schema.Message
	}

	llm, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		BaseURL:     openAIBaseURL,
		APIKey:      openAIAPIKey,
		Model:       modelName,
		Temperature: float32(0.7),
	})
	if err != nil {
		logs.Fatalf("new chat model failed: %v", err)
	}

	g := compose.NewGraph[[]*schema.Message,
		*schema.Message](compose.WithGenLocalState(func(ctx context.Context) *state {
		return &state{}
	}))
	//设置大模型节点,并且compose.WithStatePreHandler方法给全局状态机进行变更
	_ = g.AddChatModelNode("writer", llm, compose.WithStatePreHandler[[]*schema.Message, *state](func(ctx context.Context, input []*schema.Message, state *state) ([]*schema.Message, error) {
		state.currentRound++
		state.msgs = append(state.msgs, input...)
		input = append([]*schema.Message{schema.SystemMessage("you are a writer who writes jokes and revise it according to the critic's feedback. Prepend your joke with your name which is \"writer: \"")}, state.msgs...)
		return input, nil
	}), compose.WithNodeName("writer"))
	//设置林外一个大模型节点,并且compose.WithStatePreHandler方法给全局状态机进行变更
	_ = g.AddChatModelNode("critic", llm, compose.WithStatePreHandler[[]*schema.Message, *state](func(ctx context.Context, input []*schema.Message, state *state) ([]*schema.Message, error) {
		state.msgs = append(state.msgs, input...)
		input = append([]*schema.Message{schema.SystemMessage("you are a critic who ONLY gives feedback about jokes, emphasizing on funniness. Prepend your feedback with your name which is \"critic: \"")}, state.msgs...)
		return input, nil
	}), compose.WithNodeName("critic"))

	//自定义节点,将单个数据结构 转化为切片
	_ = g.AddLambdaNode("toList1", compose.ToList[*schema.Message]())
	//自定义节点,将单个数据结构 转化为切片
	_ = g.AddLambdaNode("toList2", compose.ToList[*schema.Message]())
	//图开始节点指定
	_ = g.AddEdge(compose.START, "writer")
	//分支节点,条件:当state.currentRound>=3,直接END,否则流向toList1节点
	_ = g.AddBranch("writer", compose.NewStreamGraphBranch(func(ctx context.Context, input *schema.StreamReader[*schema.Message]) (string, error) {
		input.Close()

		next := "toList1"
		if err = compose.ProcessState[*state](ctx, func(ctx context.Context, state *state) error {
			if state.currentRound >= 3 {
				next = compose.END
			}
			return nil
		}); err != nil {
			return "", err
		}

		return next, nil
	}, map[string]bool{compose.END: true, "toList1": true}))

	_ = g.AddEdge("toList1", "critic")

	_ = g.AddEdge("critic", "toList2")

	_ = g.AddEdge("toList2", "writer")

	runner, err := g.Compile(ctx)
	if err != nil {
		logs.Fatalf("compile error: %v", err)
	}

	sResponse := &streamResponse{
		ch: make(chan string),
	}
	go func() {
		for m := range sResponse.ch {
			fmt.Print(m)
		}
	}()
	handler := callbacks.NewHandlerHelper().ChatModel(&callbacks.ModelCallbackHandler{
		OnEndWithStreamOutput: sResponse.OnStreamStart,
	}).Handler()

	outStream, err := runner.Stream(ctx, []*schema.Message{schema.UserMessage("write a funny line about robot, in 20 words.")},
		compose.WithCallbacks(handler))
	if err != nil {
		logs.Fatalf("stream error: %v", err)
	}
	for {
		_, err := outStream.Recv()
		if err == io.EOF {
			close(sResponse.ch)
			break
		}
	}

	time.Sleep(5 * time.Second)
}

type streamResponse struct {
	ch chan string
}

func (s *streamResponse) OnStreamStart(ctx context.Context, runInfo *callbacks2.RunInfo, input *schema.StreamReader[*model.CallbackOutput]) context.Context {
	defer input.Close()
	s.ch <- "\n=======\n"
	for {
		frame, err := input.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			logs.Fatalf("internal error: %s\n", err)
		}

		s.ch <- frame.Message.Content
	}
	return ctx
}

func init() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}
}
