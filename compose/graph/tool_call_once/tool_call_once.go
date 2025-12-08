package main

import (
	"context"
	"errors"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/joho/godotenv"
	"likeeino/internal/logs"
	"log"
	"os"

	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/coze-dev/cozeloop-go"
)

/*
* 设置有分支节点,当发现
 */
func main() {
	openAIBaseURL := os.Getenv("OPENAI_BASE_URL")
	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	modelName := os.Getenv("OPENAI_MODEL_NAME")
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

	systemTpl := `你是一名房产经纪人，结合用户的薪酬和工作，使用 user_info API，为其提供相关的房产信息。邮箱是必须的`
	chatTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage(systemTpl),
		schema.MessagesPlaceholder("message_histories", true),
		schema.UserMessage("{query}"),
	)

	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		BaseURL:     openAIBaseURL,
		APIKey:      openAIAPIKey,
		Model:       modelName,
		Temperature: float32(0.7),
	})
	if err != nil {
		logs.Fatalf("NewChatModel failed, err=%v", err)
	}

	userInfoTool := utils.NewTool(
		&schema.ToolInfo{
			Name: "user_info",
			Desc: "根据用户的姓名和邮箱，查询用户的公司、职位、薪酬信息",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"name": {
					Type: "string",
					Desc: "用户的姓名",
				},
				"email": {
					Type: "string",
					Desc: "用户的邮箱",
				},
			}),
		},
		func(ctx context.Context, input *userInfoRequest) (output *userInfoResponse, err error) {
			//可直接查询数据库或api调用
			return &userInfoResponse{
				Name:     input.Name,
				Email:    input.Email,
				Company:  "Awesome company",
				Position: "CEO",
				Salary:   "9999",
			}, nil
		})

	info, err := userInfoTool.Info(ctx)
	if err != nil {
		logs.Fatalf("Get ToolInfo failed, err=%v", err)
	}
	//将工具方法绑定到模型上
	err = chatModel.BindTools([]*schema.ToolInfo{info})
	if err != nil {
		logs.Fatalf("BindTools failed, err=%v", err)
	}
	//创建工具节点
	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{userInfoTool},
	})
	if err != nil {
		logs.Fatalf("NewToolNode failed, err=%v", err)
	}
	//创建自定义逻辑节点
	takeOne := compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) (*schema.Message, error) {
		if len(input) == 0 {
			return nil, errors.New("input is empty")
		}
		return input[0], nil
	})

	const (
		nodeModel     = "node_model"
		nodeTools     = "node_tools"
		nodeTemplate  = "node_template"
		nodeConverter = "node_converter"
	)
	//创建分支,用于根据流输入的条件确定下一个节点。
	branch := compose.NewStreamGraphBranch(func(ctx context.Context, input *schema.StreamReader[*schema.Message]) (string, error) {
		defer input.Close()
		msg, err := input.Recv()
		if err != nil {
			return "", err
		}
		//如果输入的工具大于0,则分支直接选择工具节点
		if len(msg.ToolCalls) > 0 {
			return nodeTools, nil
		}
		//否则直接结束
		return compose.END, nil
	}, map[string]bool{compose.END: true, nodeTools: true})

	graph := compose.NewGraph[map[string]any, *schema.Message]()

	//设置添加节点
	_ = graph.AddChatTemplateNode(nodeTemplate, chatTpl)
	_ = graph.AddChatModelNode(nodeModel, chatModel)
	_ = graph.AddToolsNode(nodeTools, toolsNode)
	_ = graph.AddLambdaNode(nodeConverter, takeOne)

	_ = graph.AddEdge(compose.START, nodeTemplate)
	_ = graph.AddEdge(nodeTemplate, nodeModel)
	_ = graph.AddBranch(nodeModel, branch)
	//branch逻辑里会分成两个分支,1是直接end,2是切换到nodeTools
	_ = graph.AddEdge(nodeTools, nodeConverter)
	_ = graph.AddEdge(nodeConverter, compose.END)

	r, err := graph.Compile(ctx)
	if err != nil {
		logs.Fatalf("Compile failed, err=%v", err)
	}

	out, err := r.Invoke(ctx, map[string]any{"query": "我叫 zhangsan, 邮箱是 zhangsan@bytedance.com, 帮我推荐一处房产"})
	if err != nil {
		logs.Fatalf("Invoke failed, err=%v", err)
	}

	logs.Infof("result content: %v", out.Content)
}

type userInfoRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type userInfoResponse struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Company  string `json:"company"`
	Position string `json:"position"`
	Salary   string `json:"salary"`
}

func init() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}
}
