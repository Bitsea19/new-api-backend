package controller

import (
	"github.com/QuantumNous/new-api/setting/system_setting"
)

// paymentReturnPath builds the URL a browser is sent back to after a payment.
//
// It is a browser navigation, so it points at the frontend's origin rather
// than the API's. In the monolith both are the same and this is unchanged.
func paymentReturnPath(suffix string) string {
	return system_setting.FrontendBaseURL() + suffix
}
