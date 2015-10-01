package azure_client_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAzureClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AzureClient Suite")
}
