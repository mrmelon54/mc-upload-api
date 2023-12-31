package uploader

import (
	"bytes"
	"encoding/json"
	"fmt"
	jar_parser "github.com/mrmelon54/mc-upload-api/jar-parser"
	"io"
	"mime/multipart"
	"net/http"
)

type modrinth struct {
	conf   ModrinthConfig
	client *http.Client
}

var _ Uploader = &modrinth{}

func NewModrinthUploader(config ModrinthConfig, client *http.Client) Uploader {
	if config == (ModrinthConfig{}) {
		return &empty{}
	}
	return &modrinth{config, client}
}

type ModrinthConfig struct {
	Endpoint  string `yaml:"endpoint"`
	Token     string `yaml:"token"`
	UserAgent string `yaml:"userAgent"`
}

type modrinthUploadDataStructure struct {
	Name           string   `json:"name"`
	VersionNumber  string   `json:"version_number"`
	VersionBody    *string  `json:"version_body"`
	Dependencies   any      `json:"dependencies"`
	GameVersions   []string `json:"game_versions"`
	ReleaseChannel string   `json:"release_channel"`
	Loaders        []string `json:"loaders"`
	Featured       bool     `json:"featured"`
	ProjectId      string   `json:"project_id"`
	FileParts      []string `json:"file_parts"`
}

type modrinthUploadDataError struct {
	Error       string `json:"error"`
	Description string `json:"description"`
}

func (m *modrinth) UploadVersion(projectId string, meta jar_parser.ModMetadata, versions []string, filename string, fileBody io.Reader) (string, error) {
	bodyBuf := new(bytes.Buffer)
	mpw := multipart.NewWriter(bodyBuf)

	data := modrinthUploadDataStructure{
		Name:           filename,
		VersionNumber:  meta.VersionNumber,
		VersionBody:    nil,
		Dependencies:   []string{},
		GameVersions:   versions,
		ReleaseChannel: meta.ReleaseChannel,
		Loaders:        meta.Loaders,
		Featured:       false,
		ProjectId:      projectId,
		FileParts:      []string{"main_file"},
	}

	field, err := mpw.CreateFormField("data")
	if err != nil {
		return "", err
	}
	encoder := json.NewEncoder(field)
	if err = encoder.Encode(data); err != nil {
		return "", err
	}

	file, err := mpw.CreateFormFile("main_file", filename)
	if err != nil {
		return "", err
	}
	_, _ = io.Copy(file, fileBody)
	_ = mpw.Close()

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/version", m.conf.Endpoint), bodyBuf)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", m.conf.UserAgent)
	req.Header.Set("Authorization", m.conf.Token)
	req.Header.Add("Content-Type", mpw.FormDataContentType())

	do, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(do.Body)
	if do.StatusCode != http.StatusOK {
		var errData modrinthUploadDataError
		decoder := json.NewDecoder(do.Body)
		err := decoder.Decode(&errData)
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("modrinth remote error: %s -- %s", errData.Error, errData.Description)
	}
	var idData struct {
		Id string `json:"id"`
	}
	err = json.NewDecoder(do.Body).Decode(&idData)
	if err != nil {
		return "", err
	}
	return idData.Id, nil
}
