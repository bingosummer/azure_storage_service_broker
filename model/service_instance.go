package model

import (
	storageclient "github.com/Azure/azure-sdk-for-go/storage"
)

type ServiceInstance struct {
	Id           string `json:"id"`
	DashboardUrl string `json:"dashboard_url"`

	// The following items can be provisioned from request body
	OrganizationGuid string      `json:"organization_guid"`
	PlanId           string      `json:"plan_id"`
	ServiceId        string      `json:"service_id"`
	SpaceGuid        string      `json:"space_guid"`
	Parameters       interface{} `json:"parameters, omitempty"`

	// The following items are the allowed parameters
	ResourceGroupName   string                            `json:"resource_group_name, omitempty"`
	StorageAccountName  string                            `json:"storage_account_name, omitempty"`
	ContainerAccessType storageclient.ContainerAccessType `json:"container_access_type, omitempty"`

	// The following items are for last operations
	State       string `json:"state"`
	Description string `json:"description"`
}

type CreateServiceInstanceResponse struct {
	DashboardUrl string `json:"dashboard_url"`
}

type CreateLastOperationResponse struct {
	State       string `json:"state"`
	Description string `json:"description"`
}
