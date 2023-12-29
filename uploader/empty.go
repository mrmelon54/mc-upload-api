package uploader

import (
	jarParser "github.com/mrmelon54/mc-upload-api/jar-parser"
	"io"
)

type empty struct{}

func (e *empty) UploadVersion(_ string, _ jarParser.ModMetadata, _ []string, _ bool, _ string, _ io.Reader) (string, error) {
	return "", nil
}

var _ Uploader = &empty{}
