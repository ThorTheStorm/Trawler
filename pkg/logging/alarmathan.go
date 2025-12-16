package logging

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/k0kubun/pp"
)

// Define the Alarmathan struct and its nested structs
type Labels struct {
	AlertName   string `json:"alertname"`
	Instance    string `json:"instance"`
	Severity    string `json:"severity"`
	ServiceID   string `json:"service_id"`
	Team        string `json:"team"`
	Cluster     string `json:"cluster"`
	VarselTilOS string `json:"varseltilos"`
	App         string `json:"app"`
	Criticality string `json:"criticality"`
}

type Annotations struct {
	Description string `json:"description"`
	Summary     string `json:"summary"`
}

type Alert struct {
	Fingerprint string		`json:"fingerprint"`
	Status      string      `json:"status"`
	Labels      Labels      `json:"labels"`
	Annotations Annotations `json:"annotations"`
	StartsAt    string      `json:"startsAt"`
	EndsAt      string      `json:"endsAt"`
}

type GroupLabels struct {
	AlertName string `json:"alertname"`
}

type CommonLabels struct {
	Severity string `json:"severity"`
}

type Alarmathan struct {
	Receivers       string       `json:"receivers"`
	Status          string       `json:"status"`
	Alerts          []Alert      `json:"alerts"`
	GroupLabels     GroupLabels  `json:"groupLabels"`
	CommonLabels    CommonLabels `json:"commonLabels"`
	ExternalURL     string       `json:"externalURL"`
	Version         int          `json:"version"`
	GroupKey        string       `json:"groupKey"`
	TruncatedAlerts int          `json:"truncatedAlerts"`
}

// Send JSON to webhook
func SendToWebhook(webhookURL string, data interface{}) error {
	// Marshal the data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// pp.Printf("JSON Payload: %s\n", string(jsonData)) // Pretty print the JSON payload

	// Create the HTTP POST request
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Printf("Webhook returned status: %d", resp.StatusCode)
	}

	return nil
}

// PrintAlarm pretty prints the Alarmathan struct
func PrintAlarm(alarm Alarmathan) {
	pp.Printf("Alarm Details: %+v\n", alarm)
}

func CreateAlarm (criticality string, severity string, instance string, )
