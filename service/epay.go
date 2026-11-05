package service

import (
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

// GetCallbackAddress returns the URL the payment gateway calls back.
//
// The gateway is a server, not a browser, so this points at the API's own
// origin: the notify endpoint lives behind the backend, and a frontend URL
// would never route the gateway's request to it.
func GetCallbackAddress() string {
	if operation_setting.CustomCallbackAddress == "" {
		return system_setting.APIBaseURL()
	}
	return operation_setting.CustomCallbackAddress
}
