package awsdetail

import (
	"bytes"
	"text/template"
)

// DownloadScriptOpts options for DownloadScript.
type DownloadScriptOpts struct {
	S3Bucket       string
	S3WorldPrefix  string
	S3ServerPrefix string
}

// DownloadScript returns a script for running on an EC2 instance to download the world and server.
func DownloadScript(opts DownloadScriptOpts) string {
	funcMap := template.FuncMap{
		"toS3Path": toS3Path,
	}

	const templ = `
	set -xe
	
	# Download the world
	aws s3 cp --recursive "{{toS3Path .S3WorldPrefix}}/" "world/"
	sudo mv "world/" "/"

	# Create server directory
	aws s3 cp --recursive "{{toS3Path $.S3ServerPrefix}}/" "server/"
	sudo mv "server/" "/"
	`

	t := template.Must(template.New("download").Funcs(funcMap).Parse(templ))
	buf := &bytes.Buffer{}
	t.Execute(buf, opts)

	return buf.String()
}

// UploadScriptOpts options for UploadScript.
type UploadScriptOpts struct {
	S3Bucket       string
	S3WorldPrefix  string
	S3ServerPrefix string
	ServerFiles    []string
}

// UploadScript returns a script for running on an EC2 instance to upload the world and server.
func UploadScript(opts UploadScriptOpts) string {
	funcMap := template.FuncMap{
		"toS3Path": toS3Path,
	}

	const templ = `
	set -xe

	pushd /server
	# We use '|| true' here because some files are read-only and can't be uploaded thanks to fabric, which causes a warning
	# It seems aws s3 cp doesn't check the filter before trying to stat a thing.
	aws s3 cp --recursive "." "{{toS3Path $.S3ServerPrefix}}/" --exclude "logs/*" --exclude ".fabric/*" --exclude ".mixin.out/*" || true
	popd

	# Upload the world
	cd /world
	aws s3 cp --recursive "." "{{toS3Path $.S3WorldPrefix}}/"
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
