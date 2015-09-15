package model

type ServiceBinding struct {
	Id                string `json:"id"`
	ServiceId         string `json:"service_id"`
	AppId             string `json:"app_id"`
	ServicePlanId     string `json:"service_plan_id"`
	ServiceInstanceId string `json:"service_instance_id"`
	Credentials       Credentials
}

type CreateServiceBindingResponse struct {
	// SyslogDrainUrl string      `json:"syslog_drain_url, omitempty"`
	Credentials interface{} `json:"credentials"`
}

type Credentials struct {
	StorageAccountName string `json:"storage_account_name"`
	ContainerName      string `json:"container_name"`
	PrimaryAccessKey   string `json:"primary_access_key"`
	SecondaryAccessKey string `json:"secondary_access_key"`
}
