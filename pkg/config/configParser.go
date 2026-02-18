package config

import (
	"os"
	"time"

	yaml "gopkg.in/yaml.v3"
)

type Config struct {
	Configurations struct {
		Global struct {
			LocalStorageEnabled  bool   `yaml:"localStorageEnabled"`
			ServiceID            string `yaml:"serviceid"`
			LogLevel             string `yaml:"logLevel"`
			OutputFormat         string `yaml:"outputFormat"`
			PollIntervalMinutes  int    `yaml:"pollIntervalMinutes"`
			DataPath             string `yaml:"dataPath"`
			OnlineCrlsPath       string `yaml:"onlineCrlsPath"`
			OfflineCrlsPath      string `yaml:"offlineCrlsPath"`
			GitStoragePath       string `yaml:"gitStoragePath"`
			CAstoragePath        string `yaml:"CAstoragePath"`
			OnlineCAStoragePath  string `yaml:"onlineCAStoragePath"`
			OfflineCAStoragePath string `yaml:"offlineCAStoragePath"`
			GitRepoURL           string `yaml:"gitRepoURL"`
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
			Name         string `yaml:"name"`
			URL          string `yaml:"url"`
			CertFileName string `yaml:"certFileName"`
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

func RefreshConfig(config *Config, filePath string, intervalMinutes int) (bool, error) {
	configInfo, err := os.Stat(filePath)
	if err != nil {
		return false, err
	}

	if configInfo.ModTime().After(time.Now().Add(-time.Duration(intervalMinutes) * time.Minute)) {
		newConfig, err := ParseConfig(filePath)
		if err != nil {
			return false, err
		}
		*config = *newConfig
		return true, nil
	}

	return false, nil
}
