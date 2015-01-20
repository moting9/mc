package main

import (
	"errors"
	"hash"
	"io"
	"log"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
	"github.com/minio-io/mc/pkg/uri"
)

func parseCpOptions(c *cli.Context) (fsoptions fsOptions, err error) {
	switch len(c.Args()) {
	case 1:
		return fsOptions{}, errors.New("Missing <S3Path> or <LocalPath>")
	case 2:
		if strings.HasPrefix(c.Args().Get(0), "s3://") {
			uri := uri.ParseURI(c.Args().Get(0))
			if uri.Scheme == "" {
				return fsOptions{}, errors.New("Invalid URI scheme")
			}
			fsoptions.bucket = uri.Server
			fsoptions.key = uri.Path
			fsoptions.body = c.Args().Get(1)
			fsoptions.isget = true
			fsoptions.isput = false
		} else if strings.HasPrefix(c.Args().Get(1), "s3://") {
			uri := uri.ParseURI(c.Args().Get(1))
			if uri.Scheme == "" {
				return fsOptions{}, errors.New("Invalid URI scheme")
			}
			fsoptions.bucket = uri.Server
			fsoptions.key = c.Args().Get(0)
			fsoptions.body = c.Args().Get(0)
			fsoptions.isget = false
			fsoptions.isput = true
		}
	default:
		return fsOptions{}, errors.New("Arguments missing <S3Path> or <LocalPath>")
	}
	return
}

func doFsCopy(c *cli.Context) {
	var auth *s3.Auth
	var err error
	var bodyFile *os.File
	auth, err = getAWSEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	s3c := s3.NewS3Client(auth)

	var fsoptions fsOptions
	fsoptions, err = parseCpOptions(c)
	if err != nil {
		log.Fatal(err)
	}

	if fsoptions.isput {
		bodyFile, err = os.Open(fsoptions.body)
		defer bodyFile.Close()
		if err != nil {
			log.Fatal(err)
		}

		var bodyBuffer io.Reader
		var size int64
		var md5hash hash.Hash
		md5hash, bodyBuffer, size, err = getPutMetadata(bodyFile)
		if err != nil {
			log.Fatal(err)
		}

		err = s3c.Put(fsoptions.bucket, fsoptions.key, md5hash, size, bodyBuffer)
		if err != nil {
			log.Fatal(err)
		}
	} else if fsoptions.isget {
		var objectReader io.ReadCloser
		var objectSize int64
		bodyFile, err = os.OpenFile(fsoptions.body, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		defer bodyFile.Close()

		objectReader, objectSize, err = s3c.Get(fsoptions.bucket, fsoptions.key)
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.CopyN(bodyFile, objectReader, objectSize)
		if err != nil {
			log.Fatal(err)
		}
	}
}
