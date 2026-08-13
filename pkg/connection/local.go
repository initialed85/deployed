package connection

import (
	"github.com/initialed85/deployed/pkg/connection/connection_types"
	"github.com/initialed85/deployed/pkg/connection/local"
)

func OpenLocal(host string, port int, username string, password string) (connection_types.Deployable, error) {
	return open(host, port, username, password, local.Open, "Local")
}
