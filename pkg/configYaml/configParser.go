package configParser

import (
	"log"
	"os"

	yaml "gopkg.in/yaml.v3"
)

type Config struct {
	Configurations struct {
		Global struct {
			ServiceID           string `yaml:"serviceid"`
			LogLevel            string `yaml:"logLevel"`
			OutputFormat        string `yaml:"outputFormat"`
			PollIntervalMinutes int    `yaml:"pollIntervalMinutes"`
			OnlineCrlsPath      string `yaml:"onlineCrlspath"`
		} `yaml:"global"`
		Alarmathan struct {
			WebhookURL string `yaml:"webhookURL"`
			ServiceID  string `yaml:"serviceid"`
			Team       string `yaml:"team"`
			Cluster    string `yaml:"cluster"`
			App        string `yaml:"app"`
		} `yaml:"alarmathan"`
		OnlineCrls []struct {
			Name string `yaml:"Name"`
			URL  string `yaml:"URL"`
		} `yaml:"onlineCrls"`
	} `yaml:"configurations"`
}

func ParseConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading config file: %v\n", err)
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Printf("Error parsing config file: %v\n", err)
		return nil, err
	}

	return &config, nil
}
