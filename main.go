package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// LogMessage represents the structure of the log message
type LogMessage struct {
	Timestamp       string                 `json:"@timestamp"`
	Level           string                 `json:"@level"`
	MessageTemplate string                 `json:"@messageTemplate"`
	Fields          map[string]interface{} `json:"@fields,omitempty"`
}

// SEQLogger represents a logger that sends logs to a SEQ server
type SEQLogger struct {
	seqURL  string
	apiKey  string
	logChan chan LogMessage
}

// NewSEQLogger creates a new SEQLogger
func NewSEQLogger(seqURL, apiKey string, bufferSize int) *SEQLogger {
	logger := &SEQLogger{
		seqURL:  seqURL,
		apiKey:  apiKey,
		logChan: make(chan LogMessage, bufferSize),
	}

	go logger.processLogs()

	return logger
}

// validateLogMessage validates the structure and content of the log message
func validateLogMessage(logMessage *LogMessage) error {
	if logMessage.Timestamp == "" || logMessage.Level == "" || logMessage.MessageTemplate == "" {
		return fmt.Errorf("missing required log message fields")
	}
	return nil
}

// processLogs listens on the logChan and sends log messages to the SEQ server
func (l *SEQLogger) processLogs() {
	client := &http.Client{}

	for logMessage := range l.logChan {
		// Wrap the log message inside an "Events" array
		payload := map[string]interface{}{
			"Events": []map[string]interface{}{
				{
					"Timestamp":       logMessage.Timestamp,
					"Level":           logMessage.Level,
					"MessageTemplate": logMessage.MessageTemplate,
					"Properties":      logMessage.Fields,
				},
			},
		}

		data, err := json.Marshal(payload)
		if err != nil {
			log.Printf("Failed to marshal log message: %v", err)
			log.Printf("Local log: %s - %s", logMessage.Level, logMessage.MessageTemplate)
			continue
		}

		req, err := http.NewRequest("POST", l.seqURL, bytes.NewBuffer(data))
		if err != nil {
			log.Printf("Failed to create HTTP request: %v", err)
			log.Printf("Local log: %s - %s", logMessage.Level, logMessage.MessageTemplate)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		if l.apiKey != "" {
			req.Header.Set("X-Seq-ApiKey", l.apiKey)
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Failed to send log message: %v", err)
			log.Printf("Local log: %s - %s", logMessage.Level, logMessage.MessageTemplate)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			var responseBody bytes.Buffer
			responseBody.ReadFrom(resp.Body)
			log.Printf("SEQ server responded with %v. Response: %v", resp.Status, responseBody.String())
			log.Printf("Local log: %s - %s", logMessage.Level, logMessage.MessageTemplate)
		}
	}
}

// Log sends a log message to the logChan for processing
func (l *SEQLogger) Log(level, message string, fields map[string]interface{}) {
	logMessage := LogMessage{
		Timestamp:       time.Now().UTC().Format(time.RFC3339), // Use RFC3339 format for timestamp
		Level:           level,
		MessageTemplate: message,
		Fields:          fields,
	}

	if err := validateLogMessage(&logMessage); err != nil {
		log.Printf("Validation failed for log message: %v", err)
		log.Printf("Local log: %s - %s", logMessage.Level, logMessage.MessageTemplate)
		return
	}

	l.logChan <- logMessage
}

func main() {
	seqURL := "http://localhost:5341/api/events/raw" // SEQ server URL
	apiKey := "YourAPIKey"                           // SEQ server API key

	logger := NewSEQLogger(seqURL, apiKey, 100) // Buffer size of 100 for the log channel
	// Example usage with more logs
	logger.Log("Information", "Application started", map[string]interface{}{
		"version": "1.0.0",
	})
	// Example usage with more detailed information
	logger.Log("Error", "An error occurred", map[string]interface{}{
		"error":     "example error message",
		"userID":    "12345",
		"operation": "data processing",
		"duration":  "120ms",
		"severity":  "high",
		"details": map[string]interface{}{
			"module": "user-service",
			"method": "POST",
		},
	})

	// Allow some time for the logs to be processed before exiting
	time.Sleep(5 * time.Second)
}
