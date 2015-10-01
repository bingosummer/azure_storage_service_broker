package azure_client_test

import (
	"os"

	. "github.com/bingosummer/azure_storage_service_broker/azure_client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helpers", func() {
	var (
		credentials map[string]string
		err         error
	)

	Describe("Loading Azure credentials from environment variables", func() {
		Context("With all credentials are available", func() {
			BeforeEach(func() {
				os.Setenv("subscriptionID", "fake-subscription-id")
				os.Setenv("tenantID", "fake-tenant-id")
				os.Setenv("clientID", "fake-client-id")
				os.Setenv("clientSecret", "fake-client-secret")
				credentials, err = LoadAzureCredentials()
			})

			It("should return the credentails", func() {
				Expect(credentials["subscriptionID"]).To(Equal("fake-subscription-id"))
				Expect(credentials["tenantID"]).To(Equal("fake-tenant-id"))
				Expect(credentials["clientID"]).To(Equal("fake-client-id"))
				Expect(credentials["clientSecret"]).To(Equal("fake-client-secret"))
			})

			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("With subscriptionID is unavailable", func() {
			BeforeEach(func() {
				os.Setenv("subscriptionID", "")
				credentials, err = LoadAzureCredentials()
			})

			It("should return the zero-value for credentials", func() {
				Expect(credentials).To(BeZero())
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("With tenantID is unavailable", func() {
			BeforeEach(func() {
				os.Setenv("tenantID", "")
				credentials, err = LoadAzureCredentials()
			})

			It("should return the zero-value for credentials", func() {
				Expect(credentials).To(BeZero())
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("With clientID is unavailable", func() {
			BeforeEach(func() {
				os.Setenv("clientID", "")
				credentials, err = LoadAzureCredentials()
			})

			It("should return the zero-value for credentials", func() {
				Expect(credentials).To(BeZero())
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("With clientSecret is unavailable", func() {
			BeforeEach(func() {
				os.Setenv("clientSecret", "")
				credentials, err = LoadAzureCredentials()
			})

			It("should return the zero-value for credentials", func() {
				Expect(credentials).To(BeZero())
			})

			It("should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
