package awsdetail

import (
	"testing"
)

// Doesn't actually test much, but lets me see the rendered scripts.

func TestDownloadScript(t *testing.T) {
	_ = DownloadScript(DownloadScriptOpts{
		S3ServerPrefix: s3ServerPrefix("cliff"),
		S3WorldPrefix:  s3WorldPrefix("cliff"),
	})
}

func TestUploadScript(t *testing.T) {
	_ = UploadScript(UploadScriptOpts{
		S3ServerPrefix: s3ServerPrefix("cliff"),
		S3WorldPrefix:  s3WorldPrefix("cliff"),
	})
}

func TestStartWrapperScript(t *testing.T) {
	_ = StartWrapperScript(StartWrapperScriptOpts{
		AccountID: "12345",
		Region:    "eu-west-2",
	})
}
