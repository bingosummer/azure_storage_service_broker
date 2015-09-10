package azure_client

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/Azure/go-autorest/autorest/azure"
)

// ToJSON returns the passed item as a pretty-printed JSON string. If any JSON error occurs,
// it returns the empty string.
func ToJSON(v interface{}) string {
	j, _ := json.MarshalIndent(v, "", "  ")
	return string(j)
}

// NewServicePrincipalTokenFromCredentials creates a new ServicePrincipalToken using values of the
// passed credentials map.
func NewServicePrincipalTokenFromCredentials(c map[string]string, scope string) (*azure.ServicePrincipalToken, error) {
	return azure.NewServicePrincipalToken(c["clientID"], c["clientSecret"], c["tenantID"], scope)
}

// LoadAzureCredentials reads credentials from environment variables
func LoadAzureCredentials() (map[string]string, error) {
	credentials := make(map[string]string)

	subscriptionId := os.Getenv("subscriptionID")
	if subscriptionId == "" {
		return credentials, errors.New("No subscriptionID provided in environment variables")
	}

	tenantId := os.Getenv("tenantID")
	if tenantId == "" {
		return credentials, errors.New("No tenantID provided in environment variables")
	}

	clientId := os.Getenv("clientID")
	if clientId == "" {
		return credentials, errors.New("No clientID provided in environment variables")
	}

	clientSecret := os.Getenv("clientSecret")
	if clientSecret == "" {
		return credentials, errors.New("No clientSecret provided in environment variables")
	}

	credentials["subscriptionID"] = subscriptionId
	credentials["tenantID"] = tenantId
	credentials["clientID"] = clientId
	credentials["clientSecret"] = clientSecret

	return credentials, nil
}
