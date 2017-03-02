package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/sftp"
)

func NewS3FileHandlers() sftp.Handlers {
	sess, _ := session.NewSession(&aws.Config{
		Region:      aws.String("eu-central-1"),
		Credentials: credentials.NewSharedCredentials("", "nv"),
	})

	svc := s3.New(sess)

	handler := &fileHandler{svc}
	return sftp.Handlers{handler, handler, handler, handler}
}

type fileHandler struct {
	svc *s3.S3
}

func (f *fileHandler) Fileread(request sftp.Request) (io.ReaderAt, error) {
	return nil, os.ErrInvalid
}

func (f *fileHandler) Filewrite(request sftp.Request) (io.WriterAt, error) {
	return nil, os.ErrPermission
}

func (f *fileHandler) Filecmd(request sftp.Request) error {
	return os.ErrInvalid
}

func (f *fileHandler) Fileinfo(request sftp.Request) ([]os.FileInfo, error) {
	log.Printf("%+v", request)
	switch request.Method {
	case "List":
		return f.listDir(request.Filepath)
	case "Stat":
		l, _ := f.listDir(strings.TrimPrefix(request.Filepath, "/"))
		return nil, os.ErrInvalid
	}
	return nil, os.ErrInvalid
}

func (f *fileHandler) listDir(prefix string) ([]os.FileInfo, error) {
	if prefix == "/" {
		prefix = ""
	}
	params := &s3.ListObjectsV2Input{
		Bucket:    aws.String("nv-storage"), // Required
		Delimiter: aws.String("/"),
		Prefix:    &prefix,
	}
	resp, err := f.svc.ListObjectsV2(params)

	if err != nil {
		return nil, err
	}

	fmt.Printf("%+v\n", resp)

	// response := make([]os.FileInfo, len(resp.CommonPrefixes)+len(resp.Contents))
	response := []os.FileInfo{}
	for _, dir := range resp.CommonPrefixes {
		response = append(response, newFile(strings.TrimSuffix(*dir.Prefix, "/"), 0, time.Now(), true))
	}

	for _, f := range resp.Contents {
		response = append(response, newFile(*f.Key, *f.Size, *f.LastModified, false))
	}

	return response, nil
}

type file struct {
	name    string
	size    int64
	modTime time.Time
	isDir   bool
}

func newFile(name string, size int64, modTime time.Time, isDir bool) *file {
	return &file{
		name:    name,
		size:    size,
		modTime: modTime,
		isDir:   isDir,
	}
}

func (f *file) Name() string {
	return f.name
}

func (f *file) Size() int64 {
	return f.size
}

func (f *file) Mode() os.FileMode {
	ret := os.FileMode(0644)
	if f.isDir {
		ret = os.FileMode(0755) | os.ModeDir
	}
	return ret
}

func (f *file) ModTime() time.Time {
	return time.Now()
}

func (f *file) IsDir() bool {
	return false
}

func (f *file) Sys() interface{} {
	return &syscall.Stat_t{Uid: 0, Gid: 0}
}
