package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

var (
	force, raw, write bool
	max               int
	temp, topP        float64
	ifile             string
	sfile             string
)

func init() {
	flag.BoolVar(&raw, "r", false, "raw output")
	flag.BoolVar(&force, "f", false, "force")
	flag.BoolVar(&write, "w", false, "write")
	flag.Float64Var(&temp, "t", 0.6, "temperature")
	flag.Float64Var(&topP, "p", 1, "TopP")
	flag.IntVar(&max, "max", 4000, "max tokens")
	flag.StringVar(&ifile, "i", "", "instruction file")
	flag.StringVar(&sfile, "s", "", "source file")
	flag.Parse()
}

func main() {
	// instructions
	var instructions []string
	if ifile != "" {
		data, err := os.ReadFile(ifile)
		if err != nil {
			log.Fatal(err)
		}
		instructions = append(instructions, string(data))
	}
	instructions = append(instructions, strings.Join(flag.Args(), " "))
	if len(instructions) == 0 {
		log.Fatal("no instructions")
	}
	var code string

	if sfile != "" {
		data, err := os.ReadFile(sfile)
		if err != nil {
			log.Fatal(err)
		}
		code = string(data)
	} else {
		stat, _ := os.Stdin.Stat()
		if stat.Mode()&os.ModeCharDevice == 0 {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				log.Fatal(err)
			}
			code = string(data)
		}
	}
	output, err := EditCode(strings.Join(instructions, " "), code)
	if err != nil {
		log.Fatal(err)
	}
	if sfile != "" && write {
		if err := os.WriteFile(sfile, []byte(output), os.ModePerm); err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Print(output)
	}
}

func EditCode(instructions string, code string) (string, error) {
	// Fetch the OpenAI API key from the environment
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}
	// create prompt
	var prompt []string
	// add instructions to prompt
	prompt = append(prompt, instructions)
	prompt = append(prompt, "```", code, "```")
	// sanity check the tokens
	if tok := EstimateTokens(len(code)); tok > max {
		log.Fatalf("MaxTokens isn't large enough for the provided code: Len=%d EstimatedTokens=%d MaxTokens=%d", len(code), tok, max)
	}
	if n := EstimateTokens(len(strings.Join(prompt, " "))) + max; n > 8192 && !force {
		return "", errors.New("MaxTokens + EstimatedTokens(len(Prompt)) exceeds the model's limit of 8192")
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
							"All requests be formatted with a set of instructions followed by a fenced block of code.",
							"You will apply the instructions to the code.",
						},
						" ",
					),
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: strings.Join(prompt, "\n"),
				},
			},
			MaxTokens:   max,
			Temperature: float32(temp),
			TopP:        float32(topP),
			N:           1,
		},
	)
	if err != nil {
		return "", err
	}
	for _, c := range completions.Choices {
		if raw {
			return c.Message.Content, nil
		} else {
			return ExtractCode(c.Message.Content), nil
		}
	}
	return "", errors.New("no responses")
}

func EstimateTokens(bytes int) int {
	return int(float64(bytes) / 4)
}

func ExtractCode(s string) string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(s))
	var inFence bool
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "```") {
			if inFence {
				break
			}
			inFence = true
			continue
		}
		if inFence {
			lines = append(lines, line)
		}
	}
	// better than returning nothing
	if len(lines) == 0 {
		return s
	}
	return strings.Join(lines, "\n")
}
