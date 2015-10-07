package azure_client

import (
	"os"
	"reflect"
	"testing"
)

func TestLoadAzureCredentials(t *testing.T) {
	var (
		credentials map[string]string
		err         error
	)

	for i, test := range []struct {
		subscriptionID      string
		tenantID            string
		clientID            string
		clientSecret        string
		expectedCredentials map[string]string
		expectedErr         error
	}{
		{
			"fake-subscription-id",
			"fake-tenant-id",
			"fake-client-id",
			"fake-client-secret",
			map[string]string{
				"subscriptionID": "fake-subscription-id",
				"tenantID":       "fake-tenant-id",
				"clientID":       "fake-client-id",
				"clientSecret":   "fake-client-secret",
			},
			nil,
		},
		{
			"",
			"fake-tenant-id",
			"fake-client-id",
			"fake-client-secret",
			nil,
			ErrNotFoundSubscriptionID,
		},
		{
			"fake-subscription-id",
			"",
			"fake-client-id",
			"fake-client-secret",
			nil,
			ErrNotFoundTenantID,
		},
		{
			"fake-subscription-id",
			"fake-tenant-id",
			"",
			"fake-client-secret",
			nil,
			ErrNotFoundClientID,
		},
		{
			"fake-subscription-id",
			"fake-tenant-id",
			"fake-client-id",
			"",
			nil,
			ErrNotFoundClientSecret,
		},
	} {
		os.Setenv("subscriptionID", test.subscriptionID)
		os.Setenv("tenantID", test.tenantID)
		os.Setenv("clientID", test.clientID)
		os.Setenv("clientSecret", test.clientSecret)
		credentials, err = LoadAzureCredentials()

		if !reflect.DeepEqual(credentials, test.expectedCredentials) {
			t.Errorf("Test %d: credentials were %v but expected %v\n", i, credentials, test.expectedCredentials)
		}

		if err != test.expectedErr {
			t.Errorf("Test %d: error was %v but expected %v\n", i, err, test.expectedErr)
		}
	}
}
