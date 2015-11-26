# Cloud Foundry Service Broker for Azure Storage Service

[Cloud Foundry on Azure is generally available.](https://azure.microsoft.com/en-us/blog/general-availability-of-cloud-foundry-and-preview-access-of-pivotal-cloud-foundry/) If you want to try it, please follow [the guidance](https://github.com/cloudfoundry-incubator/bosh-azure-cpi-release/blob/master/docs/guidance.md).

[Azure Storage Service](https://azure.microsoft.com/en-us/services/storage/) offers reliable, economical cloud storage for data big and small. This broker currently publishes a single service and plan for provisioning Azure Storage Service.

## Design

The broker uses local files (should be replaced with Azure meta data service) and naming conventions to maintain the state of the services it is brokering. It does not maintain an internal database so it has no dependencies.

Capability with the Cloud Foundry service broker API is indicated by the project version number. For example, version 2.5.0 is based off the 2.5 version of the broker API.

## Creation and Naming of Azure Resources

A service provisioning call will create Azure Storage Account.

The following names are used and can be customized with a prefix:

Resource         | Name is based on     | Custom Prefix Environment Variable  | Default Prefix    | Example Name  
-----------------|----------------------|-------------------------------------|-------------------|---------------
Azure Storage Account | part of service instance ID | STORAGE_ACCOUNT_NAME_PREFIX | cf | cf2eac2d52bfc94d0faf28c0
Azure Storage Containers | service instance ID | CONTAINER_NAME_PREFIX | cloud-foundry- | cloud-foundry-2eac2d52-bfc9-4d0f-af28-c02187689d72

## Using the services in your application

### Format of Credentials

The credentials provided in a bind call have the following format:

```
"credentials":{
  "container_name": "cloud-foundry-2eac2d52-bfc9-4d0f-af28-c02187689d72",
  "primary_access_key": "PRIMARY-ACCOUNT-KEY",
  "secondary_access_key": "SECONDARY-ACCOUNT-KEY",
  "storage_account_name": "ACCOUNT-NAME"
}
```

### Demo Applications

For Python applications, you may consider using [Azure Storage Consumer](https://github.com/bingosummer/azure-storage-consumer).

In the application, you ou can use Azure SDK to operate your storage account, for e.g. put or get your blobs.

```
from azure.storage import BlobService
account_name = vcap_services[service_name][0]['credentials']['storage_account_name']
account_key = vcap_services[service_name][0]['credentials']['primary_access_key']
blob_service = BlobService(account_name, account_key)
```

## How to deploy your service and application in Cloud Foundry

1. Deploy the service broker

  1. Get the source code from Github.

    ```
    git clone https://github.com/bingosummer/azure_storage_service_broker
    cd azure_storage_service_broker
    ```

  2. Update `manifest.yml` with your credentials.

    ```
    subscriptionID: REPLACE-ME
    tenantID: REPLACE-ME
    clientID: REPLACE-ME
    clientSecret: REPLACE-ME
    authUsername: REPLACE-ME
    authPassword: REPLACE-ME
    ```

    A [service principal](https://azure.microsoft.com/en-us/documentation/articles/resource-group-create-service-principal-portal/) is composed of `tenantID`, `clientID` and `clientSecret`.

    `authUsername` and `authPassword` are the basic authentication of your service broker.

  3. Push the broker to Cloud Foundry

    ```
    cf push
    ```

2. Create a service broker

  ```
  cf create-service-broker demo-service-broker <authUsername> <authPassword> http://azure-storage-service-broker.cf.azurelovecf.com
  ```

  `<authUsername>` and `<authPassword>` should be same as the ones in `manifest.yml`.

3. Make the service public

  ```
  cf enable-service-access azurestorage
  ```

4. Show the service in the marketplace 

  ```
  cf marketplace
  ```

5. Create a service instance

  ```
  cf create-service azurestorage default myblobservice
  ```

6. Check the operation status of creating the service instance

  The creating operation is asynchronous. You can get the operation status after the creating operation.

  ```
  cf service myblobservice
  ```

7. Build the demo application

  ```
  git clone https://github.com/bingosummer/azure-storage-consumer
  cd azure-storage-consumer
  cf push --no-start
  ```

8. Bind the service instance to the application

  ```
  cf bind-service azure-storage-consumer myblobservice
  ```

9. Restart the application

  ```
  cf restart azure-storage-consumer
  ```

10. Show the service instance

  ```
  cf services
  ```

11. Get the environment variables of the application

  ```
  cf env azure-storage-consumer
  ```

12. Unbind the application from the service instance

  ```
  cf unbind-service azure-storage-consumer myblobservice
  ```

13. Delete the service instance

  ```
  cf delete-service myblobservice -f
  ```

14. Delete the service broker instance

  ```
  cf delete-service-broker demo-service-broker -f
  ```

15. Delete the service broker

  ```
  cf delete azure-storage-service-broker -f -r
  ```

16. Delete the application

  ```
  cf delete azure-storage-consumer -f -r
  ```

## More information

http://docs.cloudfoundry.org/services/
