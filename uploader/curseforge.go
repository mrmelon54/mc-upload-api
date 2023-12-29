package uploader

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MrMelon54/rescheduler"
	jar_parser "github.com/mrmelon54/mc-upload-api/jar-parser"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"time"
)

type curseforge struct {
	conf   CurseforgeConfig
	client *http.Client

	// version cache
	r         *rescheduler.Rescheduler
	cacheMu   *sync.RWMutex
	expires   time.Time
	envCache  map[string]int
	verCache  map[string]int
	platCache map[string]int
}

var _ Uploader = &curseforge{}

func NewCurseforgeUploader(config CurseforgeConfig, client *http.Client) Uploader {
	if config == (CurseforgeConfig{}) {
		return &empty{}
	}
	c := &curseforge{
		conf:      config,
		client:    client,
		cacheMu:   new(sync.RWMutex),
		envCache:  make(map[string]int),
		verCache:  make(map[string]int),
		platCache: make(map[string]int),
	}
	c.r = rescheduler.NewRescheduler(c.generateCache)
	return c
}

type CurseforgeConfig struct {
	Endpoint  string `yaml:"endpoint"`
	Token     string `yaml:"token"`
	UserAgent string `yaml:"userAgent"`
}

type CfVersionTypes struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (c *curseforge) gameVersionTypes() ([]CfVersionTypes, error) {
	req, err := http.NewRequest(http.MethodGet, c.conf.Endpoint+"/game/version-types", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.conf.UserAgent)
	req.Header.Set("X-Api-Token", c.conf.Token)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var deps []CfVersionTypes
	err = json.NewDecoder(resp.Body).Decode(&deps)
	return deps, err
}

type CfVersions struct {
	Id                int    `json:"id"`
	GameVersionTypeID int    `json:"gameVersionTypeID"`
	Name              string `json:"name"`
	Slug              string `json:"slug"`
}

func (c *curseforge) gameVersions() ([]CfVersions, error) {
	req, err := http.NewRequest(http.MethodGet, c.conf.Endpoint+"/game/versions", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.conf.UserAgent)
	req.Header.Set("X-Api-Token", c.conf.Token)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var vers []CfVersions
	err = json.NewDecoder(resp.Body).Decode(&vers)
	if err != nil {
		return nil, err
	}
	return vers, nil
}

func (c *curseforge) generateCache() {
	c.cacheMu.RLock()
	isValid := c.expires.Add(-2 * time.Hour).After(time.Now())
	c.cacheMu.RUnlock()
	if isValid {
		return
	}

	verTypes, err := c.gameVersionTypes()
	if err != nil {
		log.Println("[CF Cache] Failed to fetch game version types:", err)
		return
	}

	var envId, platId int
	verIds := make(map[int]bool)
	for _, i := range verTypes {
		switch i.Slug {
		case "environment":
			envId = i.Id
		case "modloader":
			platId = i.Id
		default:
			if strings.HasPrefix(i.Slug, "minecraft-") {
				verIds[i.Id] = true
			}
		}
	}

	versions, err := c.gameVersions()
	if err != nil {
		log.Println("[CF Cache] Failed to fetch game versions:", err)
		return
	}

	c.cacheMu.Lock()
	c.expires = time.Now().AddDate(0, 0, 1)
	mEnv := make(map[string]int)
	mPlat := make(map[string]int)
	mVer := make(map[string]int)
	for _, i := range versions {
		switch {
		case envId == i.GameVersionTypeID:
			mEnv[i.Slug] = i.Id
		case platId == i.GameVersionTypeID:
			mPlat[i.Slug] = i.Id
		case verIds[i.GameVersionTypeID]:
			mVer[i.Name] = i.Id
		}
	}
	c.envCache = mEnv
	c.platCache = mPlat
	c.verCache = mVer
	c.cacheMu.Unlock()
}

var ErrExpiredCacheData = errors.New("expired cache data")

func (c *curseforge) lookupCfIds(loaders, versions []string, environment string) ([]int, error) {
	c.r.Run()
	n := time.Now()
	c.r.Wait()
	fmt.Println(time.Since(n))
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	if c.expires.Before(time.Now()) {
		return nil, ErrExpiredCacheData
	}
	sLoaders := make([]int, len(loaders))
	sVersions := make([]int, len(versions))
	for i := range loaders {
		sLoaders[i] = c.platCache[loaders[i]]
		if sLoaders[i] == 0 {
			return nil, fmt.Errorf("invalid loader: %s", loaders[i])
		}
	}
	for i := range versions {
		sVersions[i] = c.verCache[versions[i]]
		if sVersions[i] == 0 {
			return nil, fmt.Errorf("invalid version: %s", versions[i])
		}
	}
	sEnv := make([]int, 1, 2)
	if environment == "both" || environment == "*" {
		sEnv[0] = c.envCache["client"]
		sEnv = append(sEnv, c.envCache["server"])
	} else {
		sEnv[0] = c.envCache[environment]
	}
	sLoaders = append(sLoaders, sVersions...)
	sLoaders = append(sLoaders, sEnv...)
	return sLoaders, nil
}

type curseforgeUploadDataStructure struct {
	Changelog    string `json:"changelog"`
	GameVersions []int  `json:"gameVersions"`
	ReleaseType  string `json:"releaseType"`
}

func (c *curseforge) UploadVersion(projectId string, meta jar_parser.ModMetadata, versions []string, filename string, fileBody io.Reader) (string, error) {
	intVersions, err := c.lookupCfIds(meta.Loaders, versions, meta.Environment)
	if err != nil {
		return "", fmt.Errorf("invalid game version: %w", err)
	}

	bodyBuf := new(bytes.Buffer)
	mpw := multipart.NewWriter(bodyBuf)

	data := curseforgeUploadDataStructure{
		Changelog:    "",
		GameVersions: intVersions,
		ReleaseType:  meta.ReleaseChannel,
	}

	field, err := mpw.CreateFormField("metadata")
	if err != nil {
		return "", err
	}
	encoder := json.NewEncoder(field)
	if err = encoder.Encode(data); err != nil {
		return "", err
	}

	file, err := mpw.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	_, _ = io.Copy(file, fileBody)
	_ = mpw.Close()

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/projects/%s/upload-file", c.conf.Endpoint, projectId), bodyBuf)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", c.conf.UserAgent)
	req.Header.Set("X-Api-Token", c.conf.Token)
	req.Header.Add("Content-Type", mpw.FormDataContentType())

	do, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(do.Body)
	if do.StatusCode != http.StatusOK {
		all, err := io.ReadAll(do.Body)
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("curseforge remote error: %s", string(all))
	}
	var idData struct {
		Id int `json:"id"`
	}
	err = json.NewDecoder(do.Body).Decode(&idData)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", idData.Id), nil
}
