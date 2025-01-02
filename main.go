package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

var ollamaPort = 11435

type LLMResponse struct {
	FollowsBestPractices bool   `json:"follows_best_practices"`
	Suggestions          string `json:"suggestions"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	fmt.Println("Golang-Best-Practices Hook Running!")
	files := os.Args[1:] // Files passed as arguments by pre-commit

	if len(files) == 0 {
		fmt.Println("No files provided for the hook.")
		os.Exit(0)
	}

	go startOllama(ctx)
	defer cancel()

	warn := false

	if len(files) > 20 {
		fmt.Println("Skipping as analysing more than 20 files would take too long")
		os.Exit(0)
	}

	for _, file := range files {
		if !strings.HasSuffix(file, ".go") {
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("Error reading file %s: %v\n", file, err)
			continue
		}

		llmResponse, err := queryLLM(file, string(content))
		if err != nil {
			fmt.Printf("Error querying LLM for file %s: %v\n", file, err)
			continue
		}

		if !llmResponse.FollowsBestPractices {
			warn = true
			fmt.Printf("\nFile: %s does not follow best practices:\n", file)
			fmt.Printf("Suggestions: %s\n", llmResponse.Suggestions)
		}
	}

	if warn {
		fmt.Println("\nWarning: Some files do not follow Golang best practices. Please review the suggestions above.")
	} else {
		fmt.Println("All checked files follow Golang best practices.")
	}
	os.Exit(0)
}

func startOllama(ctx context.Context) {
	cmd := exec.CommandContext(ctx, "ollama", "serve")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("OLLAMA_HOST=127.0.0.1:%d", ollamaPort))
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Format string `json:"format"`
	System string `json:"system"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

func queryLLM(filename, content string) (*LLMResponse, error) {

	llmRequest := &ollamaRequest{
		Model: "qwen2.5-coder:3b",
		System: `You check go files given for best practices following the official style guide. You will reply in json format. Only reply with the json output and nothing more. The json response should have this format:
			{
				"follows_best_practices": false,
				"suggestions": "The function name ParseYAMLConfig does not follow the Go best practices as it's repeating the package name bla bla bla..."
			}
		Suggestions need to be as short and concise as possible, there can be no suggestions if the file appears to be following the best practices.
`,
		Prompt: fmt.Sprintf("Filename: %s\nContent:\n%s", filename, content),
		Format: "json",
		Stream: false,
	}

	requestBody, err := json.Marshal(llmRequest)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %v", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/api/generate", ollamaPort), bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error running ollama command")
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body %v", err)
	}

	fmt.Println(string(body))

	oResp := &ollamaResponse{}
	err = json.Unmarshal(body, oResp)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %v", err)
	}

	fmt.Println(oResp)

	llmResponse := &LLMResponse{}

	err = json.Unmarshal([]byte(oResp.Response), llmResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling LLM response: %v", err)
	}

	return llmResponse, nil
}
