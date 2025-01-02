package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type LLMResponse struct {
	FollowsBestPractices bool   `json:"follows_best_practices"`
	Suggestions          string `json:"suggestions"`
}

func main() {

	fmt.Println("Golang-Best-Practices Hook Running!")
	files := os.Args[1:] // Files passed as arguments by pre-commit

	if len(files) == 0 {
		fmt.Println("No files provided for the hook.")
		os.Exit(0)
	}

	warn := false

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
	return
}

func queryLLM(filename, content string) (LLMResponse, error) {
	llmRequest := map[string]string{
		"filename": filename,
		"content":  content,
	}

	requestBody, err := json.Marshal(llmRequest)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("error marshalling request: %v", err)
	}

	cmd := exec.Command("ollama", "generate", "--json")
	cmd.Stdin = bytes.NewReader(requestBody)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return LLMResponse{}, fmt.Errorf("error running ollama command: %v, stderr: %s", err, stderr.String())
	}

	var llmResponse LLMResponse
	err = json.Unmarshal(stdout.Bytes(), &llmResponse)
	if err != nil {
		return LLMResponse{}, fmt.Errorf("error unmarshalling LLM response: %v", err)
	}

	return llmResponse, nil
}
