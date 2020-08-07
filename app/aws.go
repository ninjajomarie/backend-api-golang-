package external

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type AWS struct {
	session *session.Session
	log     *logrus.Entry
}

func NewAWS(log *logrus.Entry) (*AWS, error) {
	// create an AWS session which can be
	// reused if we're uploading many files
	log.Info("creating aws session")
	s, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
		Credentials: credentials.NewStaticCredentials(
			os.Getenv("AWS_KEY"),
			os.Getenv("AWS_SECRET"),
			// "secret-id",  // id
			// "secret-key", // secret
			"",
		), // token can be left blank for now
	})
	if err != nil {
		return nil, errors.Wrap(err, "creating aws sessions")
	}

	return &AWS{
		session: s,
		log:     log,
	}, nil
}

func generateS3FilePath(f *File) string {
	// sample output: users/12234/image/jpg/4/original.jpg
	// const path = `${rootFolder}/${file.userId}/${type}/${extension}/${file.id}/${name}.${file.extension}`;
	return fmt.Sprintf(
		"%s/%d/%s/%s/%d/%s.%s",
		"users",
		f.UserID.Int64,
		f.Type.String,
		f.Extension.String,
		f.ID.Int64,
		"original", // not using f.Name.String cause we will do processing later
		f.Extension.String,
	)
}

func (a *AWS) UploadFileToS3(
	fileHeader *multipart.FileHeader,
	file multipart.File,
	createdFile *File,
) (string, error) {
	// get the file size and read
	// the file content into a buffer
	size := fileHeader.Size
	buffer := make([]byte, size)
	file.Read(buffer)
	filePath := aws.String(generateS3FilePath(createdFile))

	// config settings: this is where you choose the bucket,
	// filename, content-type and storage class of the file
	// you're uploading
	a.log.Infof("uploading file to s3 - %s", *filePath)
	if _, err := s3.New(a.session).PutObject(
		&s3.PutObjectInput{
			Bucket:        aws.String(os.Getenv("AWS_BUCKET")),
			Key:           filePath,
			ACL:           aws.String(s3.BucketCannedACLPrivate), // could be private if you want it to be access by only authorized users
			Body:          bytes.NewReader(buffer),
			ContentLength: aws.Int64(int64(size)),
			ContentType:   aws.String(fileHeader.Header.Get("Content-Type")),
		},
	); err != nil {
		return "", errors.Wrap(err, "uploading file to s3")
	}

	return *filePath, nil
}
