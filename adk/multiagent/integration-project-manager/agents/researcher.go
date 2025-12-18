package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"log"
	"net/http"
	"time"
)

// 定义 web search tool 的输入和输出结构体
type webSearchInput struct {
	CurrentContext string `json:"current_context" jsonschema_description:"current context for web search"`
}
type webSearchOutput struct {
	Result []string
}

func NewResearchAgent(ctx context.Context, tcm model.ToolCallingChatModel) (adk.Agent, error) {
	//web search 的检索工具
	webSearchTool, err := utils.InferTool(
		"web_search",
		"web search tool",
		func(ctx context.Context, input *webSearchInput) (output *webSearchOutput, err error) {
			// replace it with real web search tool
			if input.CurrentContext == "" {
				return nil, fmt.Errorf("web search input is required")
			}
			//已替换为real web search
			res := duckduckToolSearch(input)
			result := make([]string, 0)
			for i := range res {
				result = append(result, res[i].Summary)
			}
			return &webSearchOutput{result}, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ResearchAgent",
		Description: "ResearchAgent负责进行研究并生成可行的解决方案。它支持中断从用户那里接收额外的上下文信息，这有助于提高研究结果的准确性和相关性。它利用网络搜索工具收集最新信息。",
		Instruction: `你是研究代理人。你的角色是:

	- 对给定的主题或问题进行彻底的研究。
	- 根据您的发现，制定可行且充分知情的解决方案。
	- 通过随时接受用户提供的其他上下文或信息来改进您的研究，从而支持中断。
	- 有效地使用网络搜索工具来收集相关和最新的数据。
	- 清晰、逻辑清晰地传达你的研究结果。
	- 如果需要提高研究质量，可以提出澄清问题。
	- 在整个互动过程中保持专业和乐于助人的语气。
	
	Tool Handling:
	- 当您认为输入信息不足以支持研究时，请使用ask_for_clarification工具要求用户补充上下文。
	- 如果上下文满足，您可以使用web_search工具从互联网获取更多数据。
	`,
		Model: tcm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{webSearchTool, newAskForClarificationTool()},
			},
		},
		MaxIterations: 5,
	})
}

type askForClarificationOptions struct {
	NewInput *string
}

func WithNewInput(input string) tool.Option {
	return tool.WrapImplSpecificOptFn(func(t *askForClarificationOptions) {
		t.NewInput = &input
	})
}

type AskForClarificationInput struct {
	Question string `json:"question" jsonschema_description:"您想向用户提出的具体问题，以获取缺失的信息"`
}

// 自定义工具,若用户提问模糊,支持中断并要求用户补充信息
func newAskForClarificationTool() tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"ask_for_clarification",
		"当用户的请求不明确或缺乏继续进行所需的信息时，调用此工具。在你有效使用其他工具之前，用它来问一个后续问题，以获得你需要的细节，比如这本书的类型。",
		func(ctx context.Context, input *AskForClarificationInput, opts ...tool.Option) (output string, err error) {
			o := tool.GetImplSpecificOptions[askForClarificationOptions](nil, opts...)
			if o.NewInput == nil {
				return "", compose.Interrupt(ctx, input.Question)
			}
			output = *o.NewInput
			o.NewInput = nil
			return output, nil
		})
	if err != nil {
		log.Fatal(err)
	}
	return t
}

// 使用duckduck 作为web search tool
func duckduckToolSearch(input *webSearchInput) []*duckduckgo.TextSearchResult {
	ctx := context.Background()
	// Create configuration
	config := &duckduckgo.Config{
		MaxResults: 3, // Limit to return 20 results
		Region:     duckduckgo.RegionWT,
		Timeout:    10 * time.Second,
	}

	// 创建自定义的 Transport，设置代理
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	// 创建自定义的 HTTP Client
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second, // 设置超时时间
	}
	// 将自定义的 HTTP Client 赋值给 Config
	config.HTTPClient = httpClient

	// Create search client
	tool, err := duckduckgo.NewTextSearchTool(ctx, config)
	if err != nil {
		log.Fatalf("NewTextSearchTool of duckduckgo failed, err=%v", err)
	}

	results := make([]*duckduckgo.TextSearchResult, 0, config.MaxResults)

	searchReq := &duckduckgo.TextSearchRequest{
		Query: input.CurrentContext,
	}
	jsonReq, err := json.Marshal(searchReq)
	if err != nil {
		log.Fatalf("Marshal of search request failed, err=%v", err)
	}

	resp, err := tool.InvokableRun(ctx, string(jsonReq))
	if err != nil {
		log.Fatalf("Search of duckduckgo failed, err=%v", err)
	}

	var searchResp duckduckgo.TextSearchResponse
	if err = json.Unmarshal([]byte(resp), &searchResp); err != nil {
		log.Fatalf("Unmarshal of search response failed, err=%v", err)
	}
	results = append(results, searchResp.Results...)
	return results
}
