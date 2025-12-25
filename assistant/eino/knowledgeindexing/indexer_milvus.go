package knowledgeindexing

import (
	"context"
	"github.com/cloudwego/eino-ext/components/indexer/milvus"
	"github.com/cloudwego/eino/components/indexer"
	cli "github.com/milvus-io/milvus-sdk-go/v2/client"
	"log"
)

func newMilvusIndexer(ctx context.Context) (idr indexer.Indexer, err error) {
	//初始化客户端
	client, err := cli.NewClient(ctx, cli.Config{
		Address: "localhost:19530",
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	MilvusCli := client

	embedder, err := newEmbedding(ctx)
	if err != nil {
		return nil, err
	}

	config := &milvus.IndexerConfig{
		Client:     MilvusCli,
		Collection: "eino_collection",
		//Fields:     fields,
		Embedding: embedder,
	}

	indexer, err := milvus.NewIndexer(ctx, config)
	if err != nil {
		return nil, err
	}
	return indexer, nil
}
