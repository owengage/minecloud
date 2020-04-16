package awsdetail

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/owengage/minecloud/pkg/minecloud"
)

// BackupWorld using the hash of the tar'd world directory.
func BackupWorld(detail *Detail, world minecloud.World) error {
	detail.Logger.Infof("backing up world %s", world)

	worldKey := s3WorldKey(string(world))

	// Kick off download of world.
	obj, err := detail.S3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s3BucketName),
		Key:    aws.String(worldKey),
	})
	if err != nil {
		return err
	}
	defer obj.Body.Close()

	// Store world in a buffer, but gzipped. Might take up a fair chunk of ram.
	// Ideally we'd stream this directly back to S3 but the S3 upload requires
	// 'Seek', which the S3 download doesn't provide on its reader.
	worldZip := &bytes.Buffer{}
	w := gzip.NewWriter(worldZip)

	// Write to the hash as well as the gzip buffer.
	teeReader := io.TeeReader(obj.Body, w)

	// Compute the hash.
	hash := sha256.New()
	_, err = io.Copy(hash, teeReader)
	if err != nil {
		return err
	}
	hexHash := hex.EncodeToString(hash.Sum(nil))

	w.Close() // Finish writing gzip

	// Check S3 to see if we already have an object stored under this hash
	// If we do we'll assume it's the exact same.
	listResult, err := detail.S3.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:  aws.String(s3BucketName),
		Prefix:  aws.String(s3BackupKey(world, hexHash)),
		MaxKeys: aws.Int64(1),
	})

	if err != nil {
		return err
	}

	if *listResult.KeyCount != 0 {
		detail.Logger.Infof("found %d exact backup of world already", *listResult.KeyCount)
		return nil
	}

	detail.Logger.Infof("uploading new version of %s", world)
	detail.Logger.Infof("hash of world %s is %s", world, hexHash)

	// Upload the new backup directly to S3 glacier
	_, err = detail.S3.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s3BucketName),
		Key:    aws.String(s3BackupKey(world, hexHash)),
		Body:   bytes.NewReader(worldZip.Bytes()),
		Metadata: map[string]*string{
			"created": aws.String(time.Now().UTC().Format(time.RFC3339)),
		},
		StorageClass: aws.String(s3.StorageClassGlacier),
	})

	if err != nil {
		return err
	}

	detail.Logger.Infof("backed up %s", world)
	return nil
}
