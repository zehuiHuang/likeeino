package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"time"

	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
)

func main() {
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
		Query: "eino",
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

	// Print results
	fmt.Println("Search Results:")
	fmt.Println("==============")
	fmt.Printf("%s\n", searchResp.Message)
	for i, result := range results {
		fmt.Printf("\n%d. Title: %s\n", i+1, result.Title)
		fmt.Printf("   URL: %s\n", result.URL)
		fmt.Printf("   Summary: %s\n", result.Summary)
	}
	fmt.Println("")
	fmt.Println("==============")
}

func init() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}
}
