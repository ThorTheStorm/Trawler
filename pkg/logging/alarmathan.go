package logging

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	config "trawler/pkg/config"

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
	Fingerprint string      `json:"fingerprint"`
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

type SeverityLevel string

const (
	SeverityLow      SeverityLevel = "Lav"
	SeverityNormal   SeverityLevel = "Middels"
	SeverityWarning  SeverityLevel = "Høy"
	SeverityCritical SeverityLevel = "Kritisk"
)

type CriticalityLevel string

const (
	CriticalityLow      CriticalityLevel = "Lav"
	CriticalityMedium   CriticalityLevel = "Middels"
	CriticalityHigh     CriticalityLevel = "Høy"
	CriticalityCritical CriticalityLevel = "Kritisk"
)

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
func PrintAlarm(alarm *Alarmathan) {
	pp.Printf("Alarm Details: %+v\n", alarm)
}

// Formats the alarm and returns a pointer to the filled alarm-object
func GenerateAlarm(config config.Config, alertName string, criticality CriticalityLevel, severity SeverityLevel, instance string, description string) *Alarmathan {

	// Generate fingerprint for the alert
	identity := fmt.Sprintf("%s:%s:%s", alertName, instance, severity)
	hash := sha256.Sum256([]byte(identity))
	fingerprint := fmt.Sprintf("%x", hash)

	// Create a alarmathan alarm
	alarm := Alarmathan{
		Receivers: "varseltilos",
		Status:    "firing",
		Alerts: []Alert{
			{
				Fingerprint: fingerprint,
				Status:      "firing",
				Labels: Labels{
					AlertName:   alertName,
					Instance:    instance,
					Severity:    string(severity),
					ServiceID:   config.Configurations.Alarmathan.ServiceID,
					Team:        config.Configurations.Alarmathan.Team,
					Cluster:     config.Configurations.Alarmathan.Cluster,
					VarselTilOS: config.Configurations.Alarmathan.VarselTilOS,
					App:         config.Configurations.Alarmathan.App,
					Criticality: string(criticality),
				},
				Annotations: Annotations{
					Description: description,
					Summary:     "",
				},
				StartsAt: "2025-12-15T12:00:00Z",
				EndsAt:   "2025-12-15T16:00:00Z",
			},
		},
		GroupLabels: GroupLabels{
			AlertName: alertName,
		},
		CommonLabels: CommonLabels{
			Severity: string(severity),
		},
		ExternalURL:     "https://nhn.no",
		Version:         4,
		GroupKey:        fmt.Sprintf("{}:{{alertname=\"%s\"}}", alertName),
		TruncatedAlerts: 0,
	} // var alarm

	return &alarm
} // func AddAlarmInfo
