package azure_client

import (
	"errors"
	"os"

	"github.com/Azure/go-autorest/autorest/azure"
)

var (
	ErrNotFoundSubscriptionID = errors.New("No subscriptionID provided in environment variables")
	ErrNotFoundTenantID       = errors.New("No tenantID provided in environment variables")
	ErrNotFoundClientID       = errors.New("No clientID provided in environment variables")
	ErrNotFoundClientSecret   = errors.New("No clientSecret provided in environment variables")
)

// NewServicePrincipalTokenFromCredentials creates a new ServicePrincipalToken using values of the
// passed credentials map.
func NewServicePrincipalTokenFromCredentials(c map[string]string, scope string) (*azure.ServicePrincipalToken, error) {
	return azure.NewServicePrincipalToken(c["clientID"], c["clientSecret"], c["tenantID"], scope)
}

// LoadAzureCredentials reads credentials from environment variables
func LoadAzureCredentials() (map[string]string, error) {
	subscriptionId := os.Getenv("subscriptionID")
	if subscriptionId == "" {
		return nil, ErrNotFoundSubscriptionID
	}

	tenantId := os.Getenv("tenantID")
	if tenantId == "" {
		return nil, ErrNotFoundTenantID
	}

	clientId := os.Getenv("clientID")
	if clientId == "" {
		return nil, ErrNotFoundClientID
	}

	clientSecret := os.Getenv("clientSecret")
	if clientSecret == "" {
		return nil, ErrNotFoundClientSecret
	}

	credentials := make(map[string]string)
	credentials["subscriptionID"] = subscriptionId
	credentials["tenantID"] = tenantId
	credentials["clientID"] = clientId
	credentials["clientSecret"] = clientSecret

	return credentials, nil
}
