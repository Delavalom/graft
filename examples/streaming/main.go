package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/delavalom/graft"
	"github.com/delavalom/graft/provider/openai"
	"github.com/delavalom/graft/stream"
)

func main() {
	model := openai.New(
		openai.WithAPIKey(os.Getenv("OPENROUTER_API_KEY")),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithModel("anthropic/claude-sonnet-4-20250514"),
	)

	agent := graft.NewAgent("assistant",
		graft.WithInstructions("You are a helpful assistant."),
	)

	runner := graft.NewDefaultRunner(model)

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		events, err := runner.RunStream(r.Context(), agent, []graft.Message{
			{Role: graft.RoleUser, Content: r.URL.Query().Get("q")},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		stream.SSEHandlerFromChannel(events).ServeHTTP(w, r)
	})

	fmt.Println("Listening on :8080")
	http.ListenAndServe(":8080", nil)
}
