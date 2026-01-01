package main

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"text/template"
)

func main() {
	//config := ark.DefaultConfig(os.Getenv("ARK_API_KEY"))
	//config.BaseURL = "https://ark.cn-beijing.volces.com/api/v3"
	//client := ark.NewClientWithConfig(config)
	//
	//fmt.Println("----- embeddings request -----")
	//req := ark.EmbeddingRequestStrings{
	//	Input: []string{
	//		"花椰菜又称菜花、花菜，是一种常见的蔬菜。",
	//	},
	//	Model:          "ep-20251203214400-dh8vt",
	//	EncodingFormat: ark.EmbeddingEncodingFormatFloat,
	//}
	//
	//resp, err := client.CreateEmbeddings(context.Background(), req)
	//if err != nil {
	//	fmt.Printf("embeddings error: %v\n", err)
	//	return
	//}
	//
	//s, _ := json.Marshal(resp)
	//fmt.Println(string(s))

	type Inventory struct {
		Material string
		Count    uint
	}
	sweaters := Inventory{"wool", 17}
	tmpl, err := template.New("test").Parse("{{.Count}} items are made of {{.Material}}")
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(os.Stdout, sweaters)
	if err != nil {
		panic(err)
	}
}

func init() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}
}
