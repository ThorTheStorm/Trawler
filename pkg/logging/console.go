package logging

import (
	"log"
	"os"
)

type LogLevel string

const (
	InfoLevel    LogLevel = "INFO"
	WarningLevel LogLevel = "WARNING"
	ErrorLevel   LogLevel = "ERROR"
	DebugLevel   LogLevel = "DEBUG"
)

type EventType string

const (
	InfoEvent    EventType = "INFO"
	WarningEvent EventType = "WARNING"
	ErrorEvent   EventType = "ERROR"
	DebugEvent   EventType = "DEBUG"
)

func LogToConsole(logLevel LogLevel, eventType EventType, message string) {
	// Create a formatted log message
	logMessage := "[" + string(eventType) + "] " + message

	// Log the message based on the log level
	switch logLevel {
	case InfoLevel:
		log.Println(logMessage)
	case WarningLevel:
		log.Println(logMessage)
	case ErrorLevel:
		log.Println(logMessage)
	case DebugLevel:
		if os.Getenv("TRAWLER_DEBUG") == "true" {
			log.Println(logMessage)
		}
	default:
		log.Println("[" + "UNKNOWN" + "] [" + string(eventType) + "] " + message)
	}
}
