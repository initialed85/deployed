package connection_types

import (
	"context"
	"io"
)

//
// interfaces for consumers of the connection abstraction
//

type Deployable interface {
	GetUsername() string
	GetConnectable() Connectable
	Close()
	RunCommand(string) (string, string, error)
	RunCommandWithSudo(string) (string, string, error)
	Upload(string, string) error
	UploadWithSudo(string, string) error
	Download(string, string) error
	DownloadWithSudo(string, string) error
}

type OpenDeployableFn func(string, int, string, string) (Deployable, error)

type RunCommandFn func(string) (string, string, error)

//
// interfaces for the implementations of the connection abstraction
//

type Connectable interface {
	GetUsername() string
	SetLocalConnection(Deployable)
	Close()
	PrepareCommand(string) (io.Reader, io.Reader, io.Writer, RunPreparedCommandFn, context.CancelFunc, error)
	UploadFile(RunCommandFn, bool, string, string) error
	UploadFolder(RunCommandFn, bool, string, string) error
	DownloadFile(RunCommandFn, bool, string, string) error
	DownloadFolder(RunCommandFn, bool, string, string) error
}

type OpenConnectableFn func(string, int, string, string) (Connectable, error)

type PrepareCommandFn func(string) (io.Reader, io.Reader, io.Writer, RunPreparedCommandFn, context.CancelFunc, error)

type RunPreparedCommandFn func() error
