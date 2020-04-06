package awsdetail

import (
	"bytes"
	"text/template"
)

// DownloadScriptOpts options for DownloadScript.
type DownloadScriptOpts struct {
	S3Bucket       string
	S3WorldKey     string
	S3ServerPrefix string
	ServerFiles    []string
}

// DownloadScript returns a script for running on an EC2 instance to download the world and server.
func DownloadScript(opts DownloadScriptOpts) string {
	if opts.ServerFiles == nil {
		opts.ServerFiles = defaultServerFiles
	}

	funcMap := template.FuncMap{
		"toS3Path": toS3Path,
	}

	const templ = `
	set -xe
	
	# Download the world
	aws s3 cp "{{toS3Path .S3WorldKey}}" world.tar
	tar xf world.tar
	rm world.tar
	sudo mv world/ /
	
	# Create server directory
	sudo mkdir /server
	sudo chown $USER:$USER /server
	cd /server

	{{range .ServerFiles}}
	aws s3 cp "{{toS3Path $.S3ServerPrefix}}/{{.}}" "{{.}}"
	{{end}}
	`

	t := template.Must(template.New("download").Funcs(funcMap).Parse(templ))
	buf := &bytes.Buffer{}
	t.Execute(buf, opts)

	return buf.String()
}

// UploadScriptOpts options for UploadScript.
type UploadScriptOpts struct {
	S3Bucket       string
	S3WorldKey     string
	S3ServerPrefix string
	ServerFiles    []string
}

// UploadScript returns a script for running on an EC2 instance to upload the world and server.
func UploadScript(opts UploadScriptOpts) string {
	if opts.ServerFiles == nil {
		opts.ServerFiles = defaultServerFiles
	}

	funcMap := template.FuncMap{
		"toS3Path": toS3Path,
	}

	const templ = `
	set -xe

	pushd /server
	{{range .ServerFiles}}
	aws s3 cp "{{.}}" "{{toS3Path $.S3ServerPrefix}}/{{.}}"
	{{end}}
	popd

	# Upload the world
	tar cf world.tar /world
	aws s3 cp world.tar "{{toS3Path .S3WorldKey}}"
	`

	t := template.Must(template.New("upload").Funcs(funcMap).Parse(templ))
	buf := &bytes.Buffer{}
	t.Execute(buf, opts)

	return buf.String()
}

// StartWrapperScriptOpts options for UploadScript.
type StartWrapperScriptOpts struct {
	AccountID string
	Region    string
}

// StartWrapperScript returns a script for running on an EC2 instance to start the server wrapper.
func StartWrapperScript(opts StartWrapperScriptOpts) string {

	const templ = `
	set -xe
	# Log in to docker
	# sed hack to remove an invalid argument, god knows why it's there.
	$(aws ecr get-login --region "{{.Region}}" | sed 's/-e none//g')
	
	docker pull "{{.AccountID}}.dkr.ecr.{{.Region}}.amazonaws.com/minecloud/server-wrapper:latest"

	docker run -d \
		--rm \
		-p 8080:80 \
		-p 25565:25565 \
		--name serverwrapper \
		--volume /server:/server \
		--volume /world:/world \
		"{{.AccountID}}.dkr.ecr.{{.Region}}.amazonaws.com/minecloud/server-wrapper:latest" \
		-world-dir /world \
		-server-dir /server
	`

	t := template.Must(template.New("wrapper").Parse(templ))
	buf := &bytes.Buffer{}
	t.Execute(buf, opts)

	return buf.String()
}

func toS3Path(key string) string {
	return "s3://" + s3BucketName + "/" + key
}

var defaultServerFiles = []string{
	"banned-ips.json",
	"banned-players.json",
	"eula.txt",
	"ops.json",
	"server.properties",
	"usercache.json",
	"whitelist.json",
}