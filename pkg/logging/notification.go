package logging

import (
	"fmt"
	cfg "trawler/pkg/config"
)

// handleErrors allows for easy handling of errors throughout the program
func HandleErrors(errChannel <-chan ErrorReport, config *cfg.Config) {
	for errReport := range errChannel {
		// Log to console
		LogToConsole(ErrorLevel, ErrorEvent,
			fmt.Sprintf("%s: %v", errReport.Context, errReport.Err))

		// Send to external endpoint
		alarm := GenerateAlarm(*config,
			errReport.Context,
			errReport.Criticality,
			errReport.Severity,
			"ThisInstanceAsItWere",
			errReport.Err.Error())

		SendToWebhook(config.Configurations.Alarmathan.WebhookURL, *alarm)
	}
}
