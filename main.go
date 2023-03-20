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
	var force bool
	var maxtokens int
	var temp, topP float64
	var filename string
	flag.BoolVar(&force, "f", false, "force")
	flag.Float64Var(&temp, "temp", 0, "temperature")
	flag.Float64Var(&topP, "top_p", 1, "TopP")
	flag.IntVar(&maxtokens, "tokens", 4000, "max tokens")
	flag.StringVar(&filename, "i", "", "instructions file")
	flag.Parse()
	// instructions
	var prompt []string
	if filename != "" {
		data, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}
		prompt = append(prompt, string(data))
	}
	prompt = append(prompt, strings.Join(flag.Args(), " "))
	if len(prompt) == 0 {
		log.Fatal("no instructions")
	}
	// Fetch the OpenAI API key from the environment
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}
	// read stdin
	prompt = append(prompt, "```")
	stat, _ := os.Stdin.Stat()
	if stat.Mode()&os.ModeCharDevice == 0 {
		code, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		prompt = append(prompt, string(code))
		// make sure we have enough tokens to at least replace the previous code
		if len(code) > maxtokens {
			log.Fatal("-tokens isn't large enough for the provided code")
		}
	}
	prompt = append(prompt, "```")
	// sanity check the tokens
	if n := len(strings.Join(prompt, " ")) + maxtokens; n > 8192 && !force {
		log.Fatal("-tokens + len(prompt) exceeds the model's limit of 8192")
	}
	// make request
	client := openai.NewClient(key)
	completions, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "gpt-4",
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleSystem,
					Content: strings.Join(
						[]string{
							"You are a programming assistant.",
							"All requests be formatted with a set of instructions followed by a fenced code block.",
							"You will apply the instructions to the code block and output the modified code without a fence around it.",
						},
						" ",
					),
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: strings.Join(prompt, "\n"),
				},
			},
			MaxTokens:   maxtokens,
			Temperature: float32(temp),
			TopP:        float32(topP),
			N:           1,
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	if len(completions.Choices) == 0 {
		log.Fatal("no responses")
	}
	for _, c := range completions.Choices {
		fmt.Print(c.Message)
		break
	}
}
