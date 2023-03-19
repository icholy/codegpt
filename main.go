package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

func main() {
	var temp, topP float64
	flag.Float64Var(&temp, "temp", 0, "temperature")
	flag.Float64Var(&topP, "top_p", 0, "TopP")
	flag.Parse()
	// Fetch the OpenAI API key from the environment
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}
	// read stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	// make request
	client := openai.NewClient(key)
	model := "code-davinci-edit-001"
	edits, err := client.Edits(
		context.Background(),
		openai.EditsRequest{
			Model:       &model,
			Input:       string(input),
			Instruction: strings.Join(flag.Args(), " "),
			N:           1,
			Temperature: float32(temp),
			TopP:        float32(topP),
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	if len(edits.Choices) == 0 {
		log.Fatal("no responses")
	}
	for _, c := range edits.Choices {
		fmt.Print(c.Text)
		break
	}
}
