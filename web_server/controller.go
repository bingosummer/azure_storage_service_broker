package web_server

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/storage"
	storageclient "github.com/Azure/azure-sdk-for-go/storage"

	ac "github.com/bingosummer/azure_storage_service_broker/azure_client"
	"github.com/bingosummer/azure_storage_service_broker/model"
	"github.com/bingosummer/azure_storage_service_broker/utils"
)

const (
	X_BROKER_API_VERSION_NAME = "X-Broker-Api-Version"
	X_BROKER_API_VERSION      = "2.5"
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

	apiVersion := r.Header.Get(X_BROKER_API_VERSION_NAME)
	supported := validateApiVersion(apiVersion, X_BROKER_API_VERSION)
	if !supported {
		fmt.Printf("API Version is %s, not supported.\n", apiVersion)
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}
	fmt.Println("API Version is " + apiVersion)

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
	instance.DashboardUrl = "http://dashbaord_url"

	err = utils.ProvisionDataFromRequest(r, &instance)
	if err != nil {
		fmt.Println("Failed to provision data from request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	acceptsIncomplete := r.URL.Query().Get("accepts_incomplete")
	if acceptsIncomplete != "true" {
		fmt.Println("Only asynchronous provisioning is supported")
		response := make(map[string]string)
		response["error"] = "AsyncRequired"
		response["description"] = "This service plan requires client support for asynchronous service operations."
		utils.WriteResponse(w, 422, response)
		return
	}

	serviceInstanceGuid := utils.ExtractVarsFromRequest(r, "service_instance_guid")
	instance.Id = serviceInstanceGuid

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

	instance.State = "in progress"
	instance.Description = "creating service instance..."

	c.instanceMap[instance.Id] = &instance
	err = utils.MarshalAndRecord(c.instanceMap, conf.DataPath, conf.ServiceInstancesFileName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := model.CreateServiceInstanceResponse{
		DashboardUrl: instance.DashboardUrl,
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
		w.WriteHeader(http.StatusGone)
		return
	}

	state, err := c.serviceClient.GetInstanceState(instance.ResourceGroupName, instance.StorageAccountName)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			w.WriteHeader(http.StatusGone)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	if state == storage.Creating || state == storage.ResolvingDNS {
		instance.State = "in progress"
		instance.Description = "Creating the service instance, state: " + string(state)
	} else if state == storage.Succeeded {
		instance.State = "succeeded"
		instance.Description = "Successfully created the service instance, state: " + string(state)
	} else {
		instance.State = "failed"
		instance.Description = "Failed to create the service instance, state: " + string(state)
	}

	err = utils.MarshalAndRecord(c.instanceMap, conf.DataPath, conf.ServiceInstancesFileName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := model.CreateLastOperationResponse{
		State:       instance.State,
		Description: instance.Description,
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

	response := make(map[string]string)
	utils.WriteResponse(w, http.StatusOK, response)
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

	response := make(map[string]string)
	utils.WriteResponse(w, http.StatusOK, response)
}

func (c *Controller) deleteAssociatedBindings(instanceId string) error {
	for id, binding := range c.bindingMap {
		if binding.ServiceInstanceId == instanceId {
			delete(c.bindingMap, id)
		}
	}

	return utils.MarshalAndRecord(c.bindingMap, conf.DataPath, conf.ServiceBindingsFileName)
}

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

func validateApiVersion(actual, expected string) bool {
	apiVersion := strings.Split(actual, ".")
	majorApiVersionActual, err1 := strconv.Atoi(apiVersion[0])
	minorApiVersionActual, err2 := strconv.Atoi(apiVersion[1])
	if err1 != nil || err2 != nil {
		return false
	}

	apiVersion = strings.Split(expected, ".")
	majorApiVersionExpected, _ := strconv.Atoi(apiVersion[0])
	minorApiVersionExpected, _ := strconv.Atoi(apiVersion[1])

	if majorApiVersionActual < majorApiVersionExpected {
		return false
	}
	if majorApiVersionActual == majorApiVersionExpected && minorApiVersionActual < minorApiVersionExpected {
		return false
	}
	return true
}
