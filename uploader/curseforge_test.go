package uploader

import (
	_ "embed"
	"encoding/json"
	"github.com/mrmelon54/rescheduler"
	"github.com/stretchr/testify/assert"
	"strings"
	"sync"
	"testing"
	"time"
)

//go:embed cf-version-types.json
var cfVersionTypesJson []byte

//go:embed cf-versions.json
var cfVersionTypes []byte

//goland:noinspection DuplicatedCode
func TestLookupCfIds(t *testing.T) {
	c := &curseforge{
		r:       rescheduler.NewRescheduler(func() {}),
		cacheMu: new(sync.RWMutex),
		expires: time.Now().Add(2 * time.Hour),
	}

	var verTypes []CfVersionTypes
	assert.NoError(t, json.Unmarshal(cfVersionTypesJson, &verTypes))

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

	var versions []CfVersions
	assert.NoError(t, json.Unmarshal(cfVersionTypes, &versions))

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

	intVersions, err := c.lookupCfIds([]string{"fabric", "quilt", "neoforge"}, []string{"1.20"}, "client")
	assert.NoError(t, err)
	assert.Len(t, intVersions, 5)
	assert.EqualValues(t, []int{7499, 9153, 10150, 9971, 9638}, intVersions)
}
