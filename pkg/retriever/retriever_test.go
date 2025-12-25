package retriever

import (
	"github.com/cloudwego/eino/schema"
	"testing"
)

func TestRetriever(t *testing.T) {

}

// GetDocuments 转化为文本
func GetDocuments(document []*schema.Document) []string {
	res := make([]string, 0)
	for i := range document {
		res = append(res, document[i].Content)
	}
	return res
}
