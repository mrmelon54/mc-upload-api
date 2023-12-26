package uploader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/MrMelon54/rescheduler"
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
	c := &curseforge{
		conf:      config,
		client:    client,
		cacheMu:   new(sync.RWMutex),
		expires:   time.Time{},
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
	req.Header.Set("X-Api-Token", c.conf.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var deps []CfVersionTypes
	err = json.NewDecoder(resp.Body).Decode(&deps)
	if err != nil {
		return nil, err
	}
	return deps, nil
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
	req.Header.Set("X-Api-Token", c.conf.Token)
	resp, err := http.DefaultClient.Do(req)
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
	c.cacheMu.Lock()
	isValid := c.expires.Add(-2 * time.Hour).After(time.Now())
	c.cacheMu.Unlock()
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

func (c *curseforge) lookupVersionId(loader, version string) (int, bool) {
	full := loader + "@" + version
	c.cacheMu.RLock()
	if c.expires.Add(-2 * time.Hour).Before(time.Now()) {
		c.cacheMu.RUnlock()
		c.r.Run()
		c.r.Wait()
		c.cacheMu.RLock()
	}
	defer c.cacheMu.RUnlock()
	id, ok := c.verCache[full]
	return id, ok
}

type curseforgeUploadDataStructure struct {
	Changelog    string `json:"changelog"`
	GameVersions []int  `json:"gameVersions"`
	ReleaseType  string `json:"releaseType"`
}

func (c *curseforge) UploadVersion(projectId, _, releaseChannel string, gameVersions, loaders []string, _ bool, filename string, fileBody io.Reader) error {
	ll := len(loaders)
	intVersions := make([]int, 0, len(gameVersions)*ll)
	for i := range loaders {
		for j := range gameVersions {
			v, ok := c.lookupVersionId(loaders[i], gameVersions[j])
			if !ok {
				return fmt.Errorf("invalid game version")
			}
			intVersions = append(intVersions, v)
		}
	}

	bodyBuf := new(bytes.Buffer)
	mpw := multipart.NewWriter(bodyBuf)

	data := curseforgeUploadDataStructure{
		Changelog:    "",
		GameVersions: intVersions,
		ReleaseType:  releaseChannel,
	}

	field, err := mpw.CreateFormField("metadata")
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(field)
	if err = encoder.Encode(data); err != nil {
		return err
	}

	file, err := mpw.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	_, _ = io.Copy(file, fileBody)
	_ = mpw.Close()

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/projects/%s/upload-file", c.conf.Endpoint, projectId), bodyBuf)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.conf.UserAgent)
	req.Header.Set("X-Api-Token", c.conf.Token)
	req.Header.Add("Content-Type", mpw.FormDataContentType())

	do, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(do.Body)
	if do.StatusCode != http.StatusOK {
		all, err := io.ReadAll(do.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("curseforge remote error: %s", string(all))
	}
	return nil
}
