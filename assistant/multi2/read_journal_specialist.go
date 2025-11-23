package multi2

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/multiagent/host"
	"github.com/cloudwego/eino/schema"
)

func readJournal(dateStr string) (io.ReadCloser, error) {
	// get today's journal file path
	filePath, err := getJournalFilePath(dateStr)
	if err != nil {
		return nil, err
	}

	// open the journal file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	// return the file as an io.ReadCloser
	return file, nil
}

func newReadJournalSpecialist(ctx context.Context) (*host.Specialist, error) {
	// create a new read journal specialist
	return &host.Specialist{
		AgentMeta: host.AgentMeta{
			Name:        "view_journal_content",
			IntendedUse: "let another agent view the content of the journal",
		},
		Streamable: func(ctx context.Context, input []*schema.Message, opts ...agent.AgentOption) (*schema.StreamReader[*schema.Message], error) {
			now := time.Now()
			dateStr := now.Format("2006-01-02")

			journal, err := readJournal(dateStr)
			if err != nil {
				return nil, err
			}

			reader, writer := schema.Pipe[*schema.Message](0)
			go func() {
				defer func() {
					if err := recover(); err != nil {
						fmt.Println("panic err:", err)
					}
				}()

				scanner := bufio.NewScanner(journal)
				scanner.Split(bufio.ScanLines)

				for scanner.Scan() {
					line := scanner.Text()
					message := &schema.Message{
						Role:    schema.Assistant,
						Content: line + "\n",
					}
					writer.Send(message, nil)
				}

				if err := scanner.Err(); err != nil {
					writer.Send(nil, err)
				}

				writer.Close()
			}()

			return reader, nil
		},
	}, nil
}
