package uploader

import (
	"io"
)

type Uploader interface {
	UploadVersion(projectId, versionNumber, releaseChannel string, gameVersions, loaders []string, featured bool, filename string, fileBody io.Reader) error
}
