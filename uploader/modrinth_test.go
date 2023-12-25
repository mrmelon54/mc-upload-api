package uploader

import (
	"bytes"
	"encoding/json"
	"github.com/mrmelon54/mc-upload-api/uploader/test"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestModrinth_UploadVersion(t *testing.T) {
	r := http.NewServeMux()
	r.HandleFunc("/version", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		mpr, err := req.MultipartReader()
		if err != nil {
			return
		}

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
	})
	srv := test.NewTestServer(r)

	m := &modrinth{
		conf:   ModrinthConfig{Token: "abcd1234"},
		client: srv,
	}
	assert.NoError(t, m.UploadVersion("123", "1.0.0-alpha", "alpha", []string{"1.20", "1.20.1"}, []string{"fabric", "forge"}, true, "name", bytes.NewReader([]byte{0x54, 0x54})))
}
