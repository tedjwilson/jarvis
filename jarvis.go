package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[91m"
	ColorGreen  = "\033[92m"
	ColorYellow = "\033[93m"
	ColorBlue   = "\033[94m"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type ChatResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

type OllamaClient struct {
	baseURL string
	client  *http.Client
}

func NewOllamaClient(baseURL string) *OllamaClient {
	return &OllamaClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 300 * time.Second},
	}
}

func (o *OllamaClient) Chat(model string, messages []Message) (string, error) {
	reqBody := ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := o.client.Post(o.baseURL+"/api/chat", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("cannot connect to Ollama. Make sure Ollama is running")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama returned status: %d", resp.StatusCode)
	}

	var fullResponse strings.Builder
	decoder := json.NewDecoder(resp.Body)

	for {
		var chatResp ChatResponse
		if err := decoder.Decode(&chatResp); err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}

		fmt.Print(chatResp.Message.Content)
		fullResponse.WriteString(chatResp.Message.Content)

		if chatResp.Done {
			break
		}
	}

	fmt.Println()
	return fullResponse.String(), nil
}

func (o *OllamaClient) TestConnection() error {
	resp, err := o.client.Get(o.baseURL + "/api/tags")
	if err != nil {
		return fmt.Errorf("cannot connect to Ollama. Make sure Ollama is running with: ollama serve")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama returned status: %d", resp.StatusCode)
	}

	return nil
}

type VoiceEngine struct {
	ttsEnabled bool
}

func NewVoiceEngine() *VoiceEngine {
	return &VoiceEngine{
		ttsEnabled: true,
	}
}

func (v *VoiceEngine) Speak(text string) {
	if !v.ttsEnabled {
		return
	}

	go func() {
		psScript := fmt.Sprintf(`Add-Type -AssemblyName System.Speech; $synth = New-Object System.Speech.Synthesis.SpeechSynthesizer; $synth.Rate = 1; $synth.Volume = 90; $synth.Speak('%s')`, 
			strings.ReplaceAll(text, "'", "''"))
		
		cmd := exec.Command("powershell", "-Command", psScript)
		cmd.Run()
	}()
}

func (v *VoiceEngine) SpeakBlocking(text string) {
	if !v.ttsEnabled {
		return
	}

	psScript := fmt.Sprintf(`Add-Type -AssemblyName System.Speech; $synth = New-Object System.Speech.Synthesis.SpeechSynthesizer; $synth.Rate = 1; $synth.Volume = 90; $synth.Speak('%s')`, 
		strings.ReplaceAll(text, "'", "''"))
	
	cmd := exec.Command("powershell", "-Command", psScript)
	cmd.Run()
}

func (v *VoiceEngine) Listen() (string, error) {
	printInfo("Listening... (speak now)")
	
	psScript := `
Add-Type -AssemblyName System.Speech
$recognizer = New-Object System.Speech.Recognition.SpeechRecognitionEngine
$recognizer.SetInputToDefaultAudioDevice()
$grammar = New-Object System.Speech.Recognition.DictationGrammar
$recognizer.LoadGrammar($grammar)
$result = $recognizer.Recognize()
if ($result -ne $null) {
    $result.Text
} else {
    ""
}
$recognizer.Dispose()
`
	
	cmd := exec.Command("powershell", "-Command", psScript)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("speech recognition error: %v", err)
	}

	text := strings.TrimSpace(string(output))
	if text == "" {
		return "", fmt.Errorf("no speech detected")
	}

	printInfo("Processing speech...")
	return text, nil
}

func printSeparator() {
	fmt.Println("\n" + strings.Repeat("-", 60) + "\n")
}

func printInfo(message string) {
	fmt.Printf("%s%s%s\n", ColorYellow, message, ColorReset)
}

func printError(message string) {
	fmt.Printf("%sError: %s%s\n", ColorRed, message, ColorReset)
}

func printUser(message string) {
	fmt.Printf("%sSir:%s %s\n", ColorGreen, ColorReset, message)
}

func printAssistant() {
	fmt.Printf("%sJarvis:%s ", ColorBlue, ColorReset)
}

func saveConversation(messages []Message) error {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("conversation_%s.txt", timestamp)

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, msg := range messages {
		if msg.Role != "system" {
			fmt.Fprintf(file, "%s: %s\n\n", strings.ToUpper(msg.Role), msg.Content)
		}
	}

	printInfo(fmt.Sprintf("Conversation saved to %s", filename))
	return nil
}

func main() {
	model := "llama2-uncensored:7b"
	client := NewOllamaClient("http://localhost:11434")
	voice := NewVoiceEngine()
	voiceInputMode := false

	conversationHistory := []Message{
		{
			Role:    "system",
			Content: "You are Jarvis, a helpful AI assistant. You are speaking with your user, whom you address as \"Sir\". Please share your thoughts on any topic you find interesting however stay on track and answer the users questions directly.",
		},
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Jarvis v1.0 - Go Edition with Voice")
	fmt.Println(strings.Repeat("=", 60) + "\n")
	printInfo("Commands: exit, quit, clear, save, mute, unmute, voice, text")
	fmt.Println()

	if err := client.TestConnection(); err != nil {
		printError(err.Error())
		fmt.Println("\nPress Enter to close...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		os.Exit(1)
	}

	voice.Speak("Jarvis online. How may I assist you, Sir?")

	reader := bufio.NewReader(os.Stdin)

	for {
		var userInput string

		if voiceInputMode {
			printInfo("Press Enter to speak (or type 'text' to switch to text input)")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if strings.ToLower(input) == "text" {
				voiceInputMode = false
				printInfo("Switched to text input mode")
				continue
			} else if strings.ToLower(input) == "exit" || strings.ToLower(input) == "quit" {
				printInfo("Goodbye!")
				voice.SpeakBlocking("Goodbye, Sir.")
				return
			} else if input != "" {
				userInput = input
			} else {
				text, err := voice.Listen()
				if err != nil {
					printError(err.Error())
					continue
				}
				userInput = text
				printUser(userInput)
			}
		} else {
			fmt.Printf("%sSir:%s ", ColorGreen, ColorReset)
			input, err := reader.ReadString('\n')
			if err != nil {
				printError(fmt.Sprintf("Error reading input: %v", err))
				continue
			}
			userInput = strings.TrimSpace(input)
		}

		if userInput == "" {
			continue
		}

		switch strings.ToLower(userInput) {
		case "exit", "quit":
			printInfo("Goodbye!")
			voice.SpeakBlocking("Goodbye, Sir.")
			return

		case "clear":
			conversationHistory = []Message{conversationHistory[0]}
			printInfo("Conversation history cleared!")
			voice.Speak("Conversation history cleared.")
			continue

		case "save":
			if err := saveConversation(conversationHistory); err != nil {
				printError(fmt.Sprintf("Could not save conversation: %v", err))
			} else {
				voice.Speak("Conversation saved.")
			}
			continue

		case "mute":
			voice.ttsEnabled = false
			printInfo("Voice output muted.")
			continue

		case "unmute":
			voice.ttsEnabled = true
			printInfo("Voice output enabled.")
			voice.Speak("Voice output enabled.")
			continue

		case "voice":
			voiceInputMode = true
			printInfo("Voice input mode enabled. Press Enter to speak.")
			voice.Speak("Voice input mode enabled.")
			continue

		case "text":
			voiceInputMode = false
			printInfo("Text input mode enabled.")
			continue
		}

		conversationHistory = append(conversationHistory, Message{
			Role:    "user",
			Content: userInput,
		})

		printAssistant()

		response, err := client.Chat(model, conversationHistory)
		if err != nil {
			printError(err.Error())
			conversationHistory = conversationHistory[:len(conversationHistory)-1]
			continue
		}

		conversationHistory = append(conversationHistory, Message{
			Role:    "assistant",
			Content: response,
		})

		voice.Speak(response)

		printSeparator()
	}
}
