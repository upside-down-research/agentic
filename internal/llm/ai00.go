package llm

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"

	//	"github.com/tidwall/gjson"
	"log"
	"net/http"
)

type AI00Server struct {
	Host string
}

type Response struct {
	Object  string `json:"object"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		Index        int `json:"index"`
		FinishReason any `json:"finish_reason"`
	} `json:"choices"`
}

func parseEvent(rawEvent string) (*Response, error) {
	const dataPrefix = "data:"
	if len(rawEvent) > len(dataPrefix) && rawEvent[:len(dataPrefix)] == dataPrefix {
		var response Response
		err := json.Unmarshal([]byte(rawEvent[len(dataPrefix):]), &response)
		if err != nil {
			return nil, err
		}
		return &response, nil
		//return rawEvent[len(dataPrefix):], nil
	}
	return nil, fmt.Errorf("invalid event format")
}
func (s AI00Server) Completion(data *LLMQuery) (string, error) {
	payloadBytes, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		panic(err)
	}

	inputBody := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/oai/chat/completions", s.Host), inputBody)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Authorization", "Bearer ai00")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", s.Host)
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Referer", s.Host)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", "Agentic 1")

	client := &http.Client{
		Transport: &http.Transport{
			// Define a custom TLSClientConfig
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Skip TLS certificate verification
			},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Process the response
	fmt.Println("Response status:", resp.Status)
	if resp.StatusCode != http.StatusOK {
		// read the entire inputBody
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)

		log.Fatalf("Unexpected response status: %s - %s", resp.Status, buf.String())
	}
	log.Println("Connected to the SSE server.")

	scanner := bufio.NewScanner(resp.Body)
	var sb strings.Builder
	for scanner.Scan() {
		data := scanner.Text()
		event, err := parseEvent(data)
		if err != nil {
			// log.Printf("Error parsing event: %v", data)
			continue
		}

		//fmt.Print(event.Choices[0].Delta.Content)
		fmt.Print(".")
		sb.WriteString(event.Choices[0].Delta.Content)

		if event.Choices[0].FinishReason == "stop" {
			break
		} else if event.Choices[0].FinishReason == nil {
			/// no op
		} else {
			log.Println("FinishReason: ", event.Choices[0].FinishReason)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading stream: %v", err)
	}
	return sb.String(), nil
}
