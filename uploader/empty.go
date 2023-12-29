package uploader

import (
	jarParser "github.com/mrmelon54/mc-upload-api/jar-parser"
	"io"
)

type empty struct{}

func (e *empty) UploadVersion(projectId string, meta jarParser.ModMetadata, versions []string, filename string, fileBody io.Reader) (string, error) {
	return "", nil
}

var _ Uploader = &empty{}
