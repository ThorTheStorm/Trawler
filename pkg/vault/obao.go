package vault

import (
	"fmt"
	"os"

	vault "github.com/hashicorp/vault/api"
)

type VaultClient = vault.Client

type VaultSecretConfig struct {
	Path string
}

var client *VaultClient

func init() {
	// Initialize Vault client and set it as the default client for the package
	defaultVaultConfig := vault.DefaultConfig()
	client, err := vault.NewClient(defaultVaultConfig)
	if err != nil {
		panic("Failed to initialize Vault client: " + err.Error())
	}

	vaultNamespace := os.Getenv("VAULT_NAMESPACE")
	if vaultNamespace == "" {
		vaultNamespace = "default"
	}
	client.SetNamespace(vaultNamespace)
}

func GetVaultClient() *VaultClient {
	return client
}

func GetSecret(crlName string) (map[string]interface{}, error) {
	secretPath := os.Getenv("VAULT_SECRET_PATH")
	if secretPath == "" {
		return nil, fmt.Errorf("VAULT_SECRET_PATH environment variable is not set")
	}

	secret, err := client.Logical().Read(secretPath)
	if err != nil {
		return nil, err
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data found at the specified Vault path: %s", secretPath)
	}

	return secret.Data, nil
}

func GetVaultSecret(secretPath string) (map[string]interface{}, error) {
	secret, err := client.Logical().Read(secretPath)
	if err != nil {
		return nil, err
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data found at the specified Vault path: %s", secretPath)
	}

	return secret.Data, nil
}
