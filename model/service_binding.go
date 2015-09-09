package model

type ServiceBinding struct {
	Id                string
	ServiceId         string
	AppId             string
	ServicePlanId     string
	ServiceInstanceId string
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
