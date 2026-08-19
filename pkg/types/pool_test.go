package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPool(t *testing.T) {
	doTest := func(user string, pass string, host string, port int) {
		parsedUser, parsedPass, parsedHost, parsedPort, err := ParseTarget(fmt.Sprintf("%s:%s@%s:%d", user, pass, host, port))
		require.NoError(t, err)
		require.Equal(t, user, parsedUser)
		require.Equal(t, pass, parsedPass)
		require.Equal(t, host, parsedHost)
		require.Equal(t, port, parsedPort)
	}

	doTest("root", "Password1", "1.2.3.4", 1234)
	doTest("root", "Password123!@#", "1.2.3.4", 1234)
}
