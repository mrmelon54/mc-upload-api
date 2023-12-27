package resolve_versions

import (
	_ "embed"
	"github.com/Masterminds/semver/v3"
	"github.com/mrmelon54/mc-upload-api/uploader/test"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

//go:embed piston-meta-manifest.json
var pistonMetaManifestJson []byte

var defaultVersions = []*semver.Version{
	semver.MustParse("1.20.4"),
	semver.MustParse("1.20.3"),
}

func TestMcVersions_gameVersions(t *testing.T) {
	v := &McVersions{
		client: &http.Client{
			Transport: test.RoundTripFunc(func(req *http.Request) *http.Response {
				rec := httptest.NewRecorder()
				rec.WriteHeader(http.StatusOK)
				rec.Write(pistonMetaManifestJson)
				return rec.Result()
			}),
		},
	}
	versions, err := v.gameVersions()
	assert.NoError(t, err)
	assert.Equal(t, len(defaultVersions), len(versions))
	for i := range versions {
		assert.True(t, versions[i].Equal(defaultVersions[i]), "%s != %s", versions[i], defaultVersions[i])
	}
}
