package azure_client

import (
	"os"
	"testing"
)

func TestLoadAzureCredentials(t *testing.T) {
	var (
		credentials map[string]string
		err         error
	)

	// With all credentials are available
	os.Setenv("subscriptionID", "fake-subscription-id")
	os.Setenv("tenantID", "fake-tenant-id")
	os.Setenv("clientID", "fake-client-id")
	os.Setenv("clientSecret", "fake-client-secret")
	credentials, err = LoadAzureCredentials()

	if credentials["subscriptionID"] != "fake-subscription-id" {
		t.Errorf("subscriptionID: got %v; want %v", credentials["subscriptionID"], "fake-subscription-id")
	}

	if credentials["tenantID"] != "fake-tenant-id" {
		t.Errorf("tenantID: got %v; want %v", credentials["tenantID"], "fake-tenant-id")
	}

	if credentials["clientID"] != "fake-client-id" {
		t.Errorf("clientID: got %v; want %v", credentials["clientID"], "fake-client-id")
	}

	if credentials["clientSecret"] != "fake-client-secret" {
		t.Errorf("clientSecret: got %v; want %v", credentials["clientSecret"], "fake-client-secret")
	}

	if err != nil {
		t.Errorf("Should not error")
	}

	// subscriptionID is unavailable
	os.Setenv("subscriptionID", "")
	credentials, err = LoadAzureCredentials()
	if credentials != nil {
		t.Errorf("should return the zero-value for credentials")
	}
	if err == nil {
		t.Errorf("Should error")
	}
	os.Setenv("subscriptionID", "fake-subscription-id")

	// tenantID is unavailable
	os.Setenv("tenantID", "")
	credentials, err = LoadAzureCredentials()
	if credentials != nil {
		t.Errorf("should return the zero-value for credentials")
	}
	if err == nil {
		t.Errorf("Should error")
	}
	os.Setenv("tenantID", "fake-tenant-id")

	// clientID is unavailable
	os.Setenv("clientID", "")
	credentials, err = LoadAzureCredentials()
	if credentials != nil {
		t.Errorf("should return the zero-value for credentials")
	}
	if err == nil {
		t.Errorf("Should error")
	}
	os.Setenv("clientID", "fake-client-id")

	// clientSecret is unavailable
	os.Setenv("clientSecret", "")
	credentials, err = LoadAzureCredentials()
	if credentials != nil {
		t.Errorf("should return the zero-value for credentials")
	}
	if err == nil {
		t.Errorf("Should error")
	}
}
