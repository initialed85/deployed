package connection

import (
	"github.com/initialed85/deployed/pkg/connection/connection_types"
	"github.com/initialed85/deployed/pkg/connection/ssh"
)

func OpenSSH(host string, port int, username string, password string) (connection_types.Deployable, error) {
	deployable, err := open(host, port, username, password, ssh.Open, "SSH")
	if err != nil {
		return nil, err
	}

	localConnection, err := OpenLocal("__local__", 0, "${USER}", "${PASS}")
	if err != nil {
		return nil, err
	}

	deployable.GetConnectable().SetLocalConnection(localConnection)

	return deployable, nil
}
