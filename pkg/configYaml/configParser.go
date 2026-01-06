package configParser

import (
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
			OnlineCrlsPath      string `yaml:"onlineCrlsPath"`
		} `yaml:"global"`
		Alarmathan struct {
			Activate    bool   `yaml:"activate"`
			WebhookURL  string `yaml:"webhookURL"`
			ServiceID   string `yaml:"serviceid"`
			Team        string `yaml:"team"`
			Cluster     string `yaml:"cluster"`
			App         string `yaml:"app"`
			VarselTilOS string `yaml:"varselTilOS"`
		} `yaml:"alarmathan"`
		OnlineCrls []struct {
			Name     string `yaml:"name"`
			URL      string `yaml:"url"`
			CertFile string `yaml:"certFile"`
		} `yaml:"onlineCrls"`
	} `yaml:"configurations"`
}

func ParseConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
