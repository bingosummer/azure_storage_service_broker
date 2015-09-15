package azure_client

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/resources"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
	storageclient "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

const (
	RESOURCE_GROUP_NAME_PREFIX  = "cloud-foundry-"
	STORAGE_ACCOUNT_NAME_PREFIX = "cf"
	CONTAINER_NAME_PREFIX       = "cloud-foundry-"
	LOCATION                    = "westus"
)

type Client interface {
	CreateInstance(instanceId string, parameters interface{}) (string, string, error)
	GetInstanceState(resourceGroupName, storageAccountName string) (storage.ProvisioningState, error)
	GetAccessKeys(resourceGroupName, storageAccountName, containerName string, containerAccessType storageclient.ContainerAccessType) (string, string, string, error)
	DeleteInstance(resourceGroupName, storageAccountName string) error
	RegenerateAccessKeys(resourceGroupName, storageAccountName string) error
}

type AzureClient struct {
	ResourceManagementClient *resources.ResourceGroupsClient
	StorageAccountsClient    *storage.StorageAccountsClient
}

func NewClient() *AzureClient {
	c, err := LoadAzureCredentials()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	spt, err := NewServicePrincipalTokenFromCredentials(c, azure.AzureResourceManagerScope)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	rmc := resources.NewResourceGroupsClient(c["subscriptionID"])
	rmc.Authorizer = spt
	rmc.PollingMode = autorest.DoNotPoll

	sac := storage.NewStorageAccountsClient(c["subscriptionID"])
	sac.Authorizer = spt
	sac.PollingMode = autorest.DoNotPoll

	return &AzureClient{
		ResourceManagementClient: &rmc,
		StorageAccountsClient:    &sac,
	}
}

func (c *AzureClient) CreateInstance(instanceId string, parameters interface{}) (string, string, error) {
	var resourceGroupName, storageAccountName, location string
	var accountType storage.AccountType

	switch parameters.(type) {
	case map[string]interface{}:
		param := parameters.(map[string]interface{})

		if param["resource_group_name"] != nil {
			resourceGroupName = param["resource_group_name"].(string)
		} else {
			resourceGroupName = RESOURCE_GROUP_NAME_PREFIX + instanceId
		}

		if param["location"] != nil {
			location = param["location"].(string)
		} else {
			location = LOCATION
		}

		if param["account_type"] != nil {
			accountType = storage.AccountType(param["account_type"].(string))
		} else {
			accountType = storage.StandardLRS
		}
	default:
		resourceGroupName = RESOURCE_GROUP_NAME_PREFIX + instanceId
		location = LOCATION
		accountType = storage.StandardLRS
	}

	err := c.createResourceGroup(resourceGroupName, location)
	if err != nil {
		fmt.Printf("Creating resource group %s failed with error:\n%v\n", resourceGroupName, err)
		return "", "", err
	}

	storageAccountName = STORAGE_ACCOUNT_NAME_PREFIX + strings.Replace(instanceId, "-", "", -1)[0:22]
	err = c.createStorageAccount(resourceGroupName, storageAccountName, location, accountType)
	if err != nil {
		fmt.Printf("Creating storage account %s.%s failed with error:\n%v\n", resourceGroupName, storageAccountName, err)
		return "", "", err
	}

	return resourceGroupName, storageAccountName, nil
}

func (c *AzureClient) GetInstanceState(resourceGroupName, storageAccountName string) (storage.ProvisioningState, error) {
	sa, err := c.StorageAccountsClient.GetProperties(resourceGroupName, storageAccountName)
	if err != nil {
		fmt.Printf("Getting instance state failed with error:\n%v\n", err)
		return "", err
	}

	return sa.Properties.ProvisioningState, nil
}

func (c *AzureClient) GetAccessKeys(instanceId, resourceGroupName, storageAccountName string, containerAccessType storageclient.ContainerAccessType) (string, string, string, error) {
	keys, err1 := c.StorageAccountsClient.ListKeys(resourceGroupName, storageAccountName)
	if err1 != nil {
		fmt.Printf("Getting access keys of %s.%s failed with error:\n%v\n", resourceGroupName, storageAccountName, err1)
		return "", "", "", err1
	}

	containerName := CONTAINER_NAME_PREFIX + instanceId
	err2 := c.createContainer(storageAccountName, keys.Key1, containerName, containerAccessType)
	if err2 != nil {
		fmt.Printf("Creating storage container %s.%s.%s failed with error:\n%v\n", resourceGroupName, storageAccountName, containerName, err2)
		return "", "", "", err2
	}

	return keys.Key1, keys.Key2, containerName, nil
}

func (c *AzureClient) DeleteInstance(resourceGroupName, storageAccountName string) error {
	r, err := c.StorageAccountsClient.Delete(resourceGroupName, storageAccountName)
	if err != nil {
		fmt.Printf("Deleting of %s.%s failed with status %s\n...%v\n", resourceGroupName, storageAccountName, r.Status, err)
		return err
	}
	fmt.Printf("Deleting of %s.%s succeeded\n", resourceGroupName, storageAccountName)
	return nil
}

func (c *AzureClient) RegenerateAccessKeys(resourceGroupName, storageAccountName string) error {
	_, err := c.StorageAccountsClient.RegenerateKey(resourceGroupName, storageAccountName,
		storage.StorageAccountRegenerateKeyParameters{
			KeyName: storage.Key1})
	if err != nil {
		fmt.Printf("Regenerating primary access key of %s.%s failed with error:\n%v\n", resourceGroupName, storageAccountName, err)
		return err
	}

	_, err = c.StorageAccountsClient.RegenerateKey(resourceGroupName, storageAccountName,
		storage.StorageAccountRegenerateKeyParameters{
			KeyName: storage.Key2})
	if err != nil {
		fmt.Printf("Regenerating secondary access key of %s.%s failed with error:\n%v\n", resourceGroupName, storageAccountName, err)
		return err
	}

	return nil
}

func (c *AzureClient) createResourceGroup(resourceGroupName, location string) error {
	rg := resources.ResourceGroup{}
	rg.Location = location

	resourceGroup, err := c.ResourceManagementClient.CreateOrUpdate(resourceGroupName, rg)
	if err != nil {
		statusCode := resourceGroup.Response.StatusCode
		if statusCode != http.StatusAccepted && statusCode != http.StatusCreated {
			fmt.Printf("Creating resource group %s failed\n", resourceGroupName)
			return err
		}
	}

	fmt.Printf("Creation initiated %s\n", resourceGroupName)
	return nil
}

func (c *AzureClient) createStorageAccount(resourceGroupName, storageAccountName, location string, accountType storage.AccountType) error {
	cna, err := c.StorageAccountsClient.CheckNameAvailability(
		storage.StorageAccountCheckNameAvailabilityParameters{
			Name: storageAccountName,
			Type: "Microsoft.Storage/storageAccounts"})
	if err != nil {
		fmt.Printf("Error: %v", err)
		return err
	}
	if !cna.NameAvailable {
		fmt.Printf("%s is unavailable -- try again\n", storageAccountName)
		return errors.New(storageAccountName + " is unavailable")
	}
	fmt.Printf("Storage account name %s is available\n", storageAccountName)

	cp := storage.StorageAccountCreateParameters{}
	cp.Location = location
	cp.Properties.AccountType = accountType

	sa, err := c.StorageAccountsClient.Create(resourceGroupName, storageAccountName, cp)
	if err != nil {
		if sa.Response.StatusCode != http.StatusAccepted {
			fmt.Printf("Creation of %s.%s failed", resourceGroupName, storageAccountName)
			return err
		}
	}

	fmt.Printf("Creation initiated %s.%s\n", resourceGroupName, storageAccountName)
	return nil
}

func (c *AzureClient) createContainer(storageAccountName, primaryAccessKey, containerName string, containerAccessType storageclient.ContainerAccessType) error {
	storageClient, err1 := storageclient.NewBasicClient(storageAccountName, primaryAccessKey)
	if err1 != nil {
		fmt.Println("Creating storage client failed")
		return err1
	}

	blobStorageClient := storageClient.GetBlobService()
	ok, err2 := blobStorageClient.CreateContainerIfNotExists(containerName, containerAccessType)
	if err2 != nil {
		fmt.Println("Creating storage container failed")
		return err2
	}
	if !ok {
		fmt.Println("Storage container already existed")
	}

	return nil
}
