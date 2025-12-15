package main

import (
	"encoding/asn1"
	"log"
	"time"
	configParser "trawler/pkg/configYaml"
	. "trawler/pkg/crl"
	alarmathan "trawler/pkg/logging"
)

type crlTimeStamps struct {
	ThisUpdate     time.Time `json:"thisUpdate"`
	NextUpdate     time.Time `json:"nextUpdate"`
	NextCRLPublish time.Time `json:"nextPublish"`
}

func main() {

	// Import configurations from config file
	config, err := configParser.ParseConfig("./config.yaml")
	if err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	// Create a sample alarmathan alarm
	var alarm = alarmathan.Alarmathan{
		Receivers: "varseltilos",
		Status:    "firing",
		Alerts: []alarmathan.Alert{
			{
				Status: "firing",
				Labels: alarmathan.Labels{
					AlertName:   config.Configurations.Alarmathan.App + "-Test-Alert",
					Instance:    "toregil-01",
					Severity:    "Normal",
					ServiceID:   config.Configurations.Alarmathan.ServiceID,
					Team:        config.Configurations.Alarmathan.Team,
					Cluster:     config.Configurations.Alarmathan.Cluster,
					VarselTilOS: "test",
					App:         "trawler",
					Criticality: "Kritisk",
				},
				Annotations: alarmathan.Annotations{
					Description: "This is the trawler, reporting for duty.",
					Summary:     "o7 o7 o7",
				},
				StartsAt: "2025-12-15T12:00:00Z",
				EndsAt:   "2025-12-15T16:00:00Z",
			},
		},
		GroupLabels: alarmathan.GroupLabels{
			AlertName: "TEST-ALERT",
		},
		CommonLabels: alarmathan.CommonLabels{
			Severity: "critical",
		},
		ExternalURL:     "https://nhn.no",
		Version:         4,
		GroupKey:        "{}:{{alertname=\"TestAlert\"}}",
		TruncatedAlerts: 0,
	} // var alarm

	// Send the alarm to the webhook
	webhookURL := config.Configurations.Alarmathan.WebhookURL
	err = alarmathan.SendToWebhook(webhookURL, alarm)
	if err != nil {
		log.Printf("Error sending to webhook: %v\n", err)
	} else {
		log.Println("Alarm sent to webhook successfully.")
	}

	// Define the OID for the Microsoft-specific "Next CRL Publish" extension
	var NEXT_PUBLISH_OID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 21, 4}

	// Define where to retrieve the CRL from
	//crlUrl := "http://crl4.digicert.com/DigiCertEVRSACAG2.crl"

	// Loop through all CRL URLs defined in the config file
	for i := 0; i < len(config.Configurations.OnlineCrls); i++ {

		crlUrl := config.Configurations.OnlineCrls[i].URL // Get the URL from the config file

		// Read out the raw CRL data from the crl retrieved from the above URL
		rawCRL, err := RetrieveCertificateRevocationList(crlUrl)
		if err != nil {
			log.Printf("Error retrieving CRL: %v\n", err)
			return
		}

		// Parse the raw CRL data into a structured format from ASN.1 DER
		decodedCRL, err := ParseCertificateRevocationList(rawCRL)
		if err != nil {
			log.Printf("Error parsing CRL: %v\n", err)
			return
		}
		crl := decodedCRL

		// Extract the Next CRL Publish time from the Microsoft-specific extension
		var nextPublishTime time.Time
		nextCRLPublish := FindExtension(crl.Extensions, NEXT_PUBLISH_OID)
		if nextCRLPublish != nil {
			_, err := asn1.Unmarshal(nextCRLPublish.Value, &nextPublishTime)
			if err != nil {
				log.Printf("Error unmarshaling Next CRL Publish time: %v\n", err)
			}
		} else {
			log.Printf("Next CRL Publish extension not found.\n")
		}

		// Prepare the published values for output/usage
		if nextPublishTime.IsZero() {
			nextPublishTime = time.Time{} // Set to zero value if not found
		}
		// crlTimeStamps := crlTimeStamps{
		// 	ThisUpdate:     crl.ThisUpdate,
		// 	NextUpdate:     crl.NextUpdate,
		// 	NextCRLPublish: nextPublishTime, // This is a ADCS (Microsoft) specific field and not part of the standard x509.RevocationList
		// }

		// log.Printf("crl: %+v\n", crl)
		// pp.Printf("CRL Published Values: %+v\n", crlTimeStamps)                   // Pretty print the CRL timestamps
		// pp.Printf("Is NextCRLPublish Zero Value? %v\n", nextPublishTime.IsZero()) // Check and print if NextCRLPublish is zero value

	} // for i, crlUrl := range config.Configuration.OnlineCrls.URL
}
