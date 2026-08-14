package env

import (
	"os"
	"testing"

	"github.com/initialed85/deployed/pkg/helpers/pointers"
)

var IsDebug = os.Getenv("DEBUG") == "1" || testing.Testing() == true

var ForceWithSudo *bool

func init() {
	if os.Getenv("FORCE_WITH_SUDO") == "1" {
		ForceWithSudo = pointers.Ptr(true)
	} else if os.Getenv("FORCE_WITH_SUDO") == "0" {
		ForceWithSudo = pointers.Ptr(false)
	}
}
