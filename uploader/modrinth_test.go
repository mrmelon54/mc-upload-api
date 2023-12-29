package uploader

import (
	"bytes"
	"encoding/json"
	jar_parser "github.com/mrmelon54/mc-upload-api/jar-parser"
	"github.com/mrmelon54/mc-upload-api/uploader/test"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestModrinth_UploadVersion(t *testing.T) {
	r := http.NewServeMux()
	r.HandleFunc("/version", func(rw http.ResponseWriter, req *http.Request) {
		mpr, err := req.MultipartReader()
		assert.NoError(t, err)

		dataPart, err := mpr.NextPart()
		assert.NoError(t, err)
		assert.Equal(t, "data", dataPart.FormName())

		var jData struct {
			Name          string   `json:"name"`
			VersionNumber string   `json:"version_number"`
			Dependencies  []any    `json:"dependencies"`
			GameVersions  []string `json:"game_versions"`
			VersionType   string   `json:"version_type"`
			Loaders       []string `json:"loaders"`
			Featured      bool     `json:"featured"`
			ProjectId     string   `json:"project_id"`
			FileParts     []string `json:"file_parts"`
		}

		assert.NoError(t, json.NewDecoder(dataPart).Decode(&jData))

		for _, filePartName := range jData.FileParts {
			filePart, err := mpr.NextPart()
			assert.NoError(t, err)
			assert.Equal(t, filePartName, filePart.FormName())
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"id":"123aaa"}`))
	})
	srv := test.NewTestServer(r)

	m := &modrinth{
		conf:   ModrinthConfig{Token: "abcd1234"},
		client: srv,
	}
	mrId, err := m.UploadVersion("123", jar_parser.ModMetadata{
		VersionNumber:  "1.0.0",
		ReleaseChannel: "alpha",
		GameVersions:   nil,
		Loaders:        []string{"fabric", "forge"},
	}, []string{"1.20", "1.20.1"}, "my-test-file.jar", bytes.NewReader([]byte{0x54, 0x54}))
	assert.NoError(t, err)
	println("mrId:", mrId)
}
