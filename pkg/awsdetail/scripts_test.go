package awsdetail

import (
	"testing"
)

// Doesn't actually test much, but lets me see the rendered scripts.

func TestDownloadScript(t *testing.T) {
	_ = DownloadScript(DownloadScriptOpts{
		S3ServerPrefix: s3ServerPrefix("cliff"),
		S3WorldKey:     s3WorldKey("cliff"),
		ServerFiles: []string{
			"banned-ips.json",
			"banned-players.json",
			"eula.txt",
			"ops.json",
			"server.properties",
			"usercache.json",
			"whitelist.json",
		},
	})
}

func TestUploadScript(t *testing.T) {
	_ = UploadScript(UploadScriptOpts{
		S3ServerPrefix: s3ServerPrefix("cliff"),
		S3WorldKey:     s3WorldKey("cliff"),
	})
}

func TestStartWrapperScript(t *testing.T) {
	_ = StartWrapperScript(StartWrapperScriptOpts{
		AccountID: "12345",
		Region:    "eu-west-2",
	})
}
