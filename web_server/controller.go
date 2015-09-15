package web_server

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/arm/storage"
	storageclient "github.com/Azure/azure-sdk-for-go/storage"

	ac "github.com/bingosummer/azure_storage_service_broker/azure_client"
	"github.com/bingosummer/azure_storage_service_broker/model"
	"github.com/bingosummer/azure_storage_service_broker/utils"
)

const (
	DEFAULT_POLLING_INTERVAL_SECONDS = 10
)

type Controller struct {
	serviceClient ac.Client

	instanceMap map[string]*model.ServiceInstance
	bindingMap  map[string]*model.ServiceBinding
}

func NewController(instanceMap map[string]*model.ServiceInstance, bindingMap map[string]*model.ServiceBinding) *Controller {
	serviceClient := ac.NewClient()
	if serviceClient == nil {
		return nil
	}

	return &Controller{
		instanceMap:   instanceMap,
		bindingMap:    bindingMap,
		serviceClient: serviceClient,
	}
}

func (c *Controller) Catalog(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Get Service Broker Catalog...")

	statusCode, err := authentication(r)
	if err != nil {
		w.WriteHeader(statusCode)
		return
	}

	var catalog model.Catalog
	err = utils.ReadAndUnmarshal(&catalog, conf.CatalogPath, "catalog.json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	utils.WriteResponse(w, http.StatusOK, catalog)
}

func (c *Controller) CreateServiceInstance(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Create Service Instance...")

	statusCode, err := authentication(r)
	if err != nil {
		w.WriteHeader(statusCode)
		return
	}

	var instance model.ServiceInstance

	err = utils.ProvisionDataFromRequest(r, &instance)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	serviceInstanceGuid := utils.ExtractVarsFromRequest(r, "service_instance_guid")

	var containerAccessType storageclient.ContainerAccessType
	switch instance.Parameters.(type) {
	case map[string]interface{}:
		param := instance.Parameters.(map[string]interface{})

		if param["container_access_type"] != nil {
			containerAccessType = storageclient.ContainerAccessType(param["container_access_type"].(string))
		} else {
			containerAccessType = storageclient.ContainerAccessTypePrivate
		}
	default:
		containerAccessType = storageclient.ContainerAccessTypePrivate
	}

	resourceGroupName, storageAccountName, err := c.serviceClient.CreateInstance(serviceInstanceGuid, instance.Parameters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	instance.ResourceGroupName = resourceGroupName
	instance.StorageAccountName = storageAccountName
	instance.ContainerAccessType = containerAccessType
	instance.DashboardUrl = "http://dashbaord_url"
	instance.Id = serviceInstanceGuid
	instance.LastOperation = &model.LastOperation{
		State:                    "in progress",
		Description:              "creating service instance...",
		AsyncPollIntervalSeconds: DEFAULT_POLLING_INTERVAL_SECONDS,
	}

	c.instanceMap[instance.Id] = &instance
	err = utils.MarshalAndRecord(c.instanceMap, conf.DataPath, conf.ServiceInstancesFileName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := model.CreateServiceInstanceResponse{
		DashboardUrl:  instance.DashboardUrl,
		LastOperation: instance.LastOperation,
	}
	utils.WriteResponse(w, http.StatusAccepted, response)
}

func (c *Controller) GetServiceInstance(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Get Service Instance State....")

	statusCode, err := authentication(r)
	if err != nil {
		w.WriteHeader(statusCode)
		return
	}

	instanceId := utils.ExtractVarsFromRequest(r, "service_instance_guid")
	instance := c.instanceMap[instanceId]
	if instance == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	state, err := c.serviceClient.GetInstanceState(instance.ResourceGroupName, instance.StorageAccountName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if state == storage.Creating || state == storage.ResolvingDNS {
		instance.LastOperation.State = "in progress"
		instance.LastOperation.Description = "creating service instance..."
	} else if state == storage.Succeeded {
		instance.LastOperation.State = "succeeded"
		instance.LastOperation.Description = "successfully created service instance"
	} else {
		instance.LastOperation.State = "failed"
		instance.LastOperation.Description = "failed to create service instance"
	}

	err = utils.MarshalAndRecord(c.instanceMap, conf.DataPath, conf.ServiceInstancesFileName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := model.CreateServiceInstanceResponse{
		DashboardUrl:  instance.DashboardUrl,
		LastOperation: instance.LastOperation,
	}
	utils.WriteResponse(w, http.StatusOK, response)
}

func (c *Controller) RemoveServiceInstance(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Remove Service Instance...")

	statusCode, err := authentication(r)
	if err != nil {
		w.WriteHeader(statusCode)
		return
	}

	instanceId := utils.ExtractVarsFromRequest(r, "service_instance_guid")
	instance := c.instanceMap[instanceId]
	if instance == nil {
		w.WriteHeader(http.StatusGone)
		return
	}

	err = c.serviceClient.DeleteInstance(instance.ResourceGroupName, instance.StorageAccountName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	delete(c.instanceMap, instanceId)
	utils.MarshalAndRecord(c.instanceMap, conf.DataPath, conf.ServiceInstancesFileName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = c.deleteAssociatedBindings(instanceId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	utils.WriteResponse(w, http.StatusOK, "{}")
}

func (c *Controller) Bind(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Bind Service Instance...")

	statusCode, err := authentication(r)
	if err != nil {
		w.WriteHeader(statusCode)
		return
	}

	bindingId := utils.ExtractVarsFromRequest(r, "service_binding_guid")
	instanceId := utils.ExtractVarsFromRequest(r, "service_instance_guid")

	instance := c.instanceMap[instanceId]
	if instance == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	primaryAccessKey, secondaryAccessKey, containerName, err := c.serviceClient.GetAccessKeys(instanceId, instance.ResourceGroupName, instance.StorageAccountName, instance.ContainerAccessType)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	credentials := model.Credentials{
		StorageAccountName: instance.StorageAccountName,
		ContainerName:      containerName,
		PrimaryAccessKey:   primaryAccessKey,
		SecondaryAccessKey: secondaryAccessKey,
	}

	response := model.CreateServiceBindingResponse{
		Credentials: credentials,
	}

	c.bindingMap[bindingId] = &model.ServiceBinding{
		Id:                bindingId,
		ServiceId:         instance.ServiceId,
		ServicePlanId:     instance.PlanId,
		ServiceInstanceId: instance.Id,
		Credentials:       credentials,
	}

	err = utils.MarshalAndRecord(c.bindingMap, conf.DataPath, conf.ServiceBindingsFileName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	utils.WriteResponse(w, http.StatusCreated, response)
}

func (c *Controller) UnBind(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Unbind Service Instance...")

	statusCode, err := authentication(r)
	if err != nil {
		w.WriteHeader(statusCode)
		return
	}

	bindingId := utils.ExtractVarsFromRequest(r, "service_binding_guid")
	instanceId := utils.ExtractVarsFromRequest(r, "service_instance_guid")
	instance := c.instanceMap[instanceId]
	if instance == nil {
		w.WriteHeader(http.StatusGone)
		return
	}

	err = c.serviceClient.RegenerateAccessKeys(instance.ResourceGroupName, instance.StorageAccountName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	delete(c.bindingMap, bindingId)
	err = utils.MarshalAndRecord(c.bindingMap, conf.DataPath, conf.ServiceBindingsFileName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	utils.WriteResponse(w, http.StatusOK, "{}")
}

func (c *Controller) deleteAssociatedBindings(instanceId string) error {
	for id, binding := range c.bindingMap {
		if binding.ServiceInstanceId == instanceId {
			delete(c.bindingMap, id)
		}
	}

	return utils.MarshalAndRecord(c.bindingMap, conf.DataPath, conf.ServiceBindingsFileName)
}

// Private
func authentication(r *http.Request) (int, error) {
	authUsername, authPassword, err := loadAuthCredentials()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	username, password, ok := r.BasicAuth()
	if !ok {
		return http.StatusUnauthorized, errors.New("No username and password provided in the request's Authorization header")
	}

	if username != authUsername || password != authPassword {
		return http.StatusUnauthorized, errors.New("The username and password are invalid")
	}

	return 0, nil
}

func loadAuthCredentials() (string, string, error) {
	username := os.Getenv("authUsername")
	if username == "" {
		return "", "", errors.New("No auth_username provided in environment variables")
	}

	password := os.Getenv("authPassword")
	if password == "" {
		return "", "", errors.New("No auth_password provided in environment variables")
	}

	return username, password, nil
}
