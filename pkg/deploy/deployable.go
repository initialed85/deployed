package deploy

type Deployable interface {
	Close()
	RunCommand(string) (string, string, error)
	RunCommandWithSudo(string) (string, string, error)
	Upload(string, string) error
	UploadWithSudo(string, string) error
	Download(string, string) error
	DownloadWithSudo(string, string) error
}
