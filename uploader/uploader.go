package uploader

import (
	jar_parser "github.com/mrmelon54/mc-upload-api/jar-parser"
	"io"
)

type Uploader interface {
	UploadVersion(projectId string, meta jar_parser.ModMetadata, versions []string, featured bool, filename string, fileBody io.Reader) (string, error)
}
