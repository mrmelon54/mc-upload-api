package resolve_versions

import (
	"encoding/json"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/MrMelon54/rescheduler"
	"log"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

const McVersionManifest = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"

func toMcVersion(ver *semver.Version) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprint(ver.Major()))
	sb.WriteByte('.')
	sb.WriteString(fmt.Sprint(ver.Minor()))
	if ver.Patch() > 0 {
		sb.WriteByte('.')
		sb.WriteString(fmt.Sprint(ver.Patch()))
	}
	return sb.String()
}

type McVersions struct {
	client *http.Client

	// version cache
	r        *rescheduler.Rescheduler
	cacheMu  *sync.RWMutex
	expires  time.Time
	versions []*semver.Version
}

func NewMcVersionCache(client *http.Client) *McVersions {
	v := &McVersions{
		client:  client,
		cacheMu: new(sync.RWMutex),
	}
	v.r = rescheduler.NewRescheduler(v.generateCache)
	return v
}

type PistonMetaManifest struct {
	Latest struct {
		Release  string `json:"release"`
		Snapshot string `json:"snapshot"`
	} `json:"latest"`
	Versions []struct {
		Id              string    `json:"id"`
		Type            string    `json:"type"`
		Url             string    `json:"url"`
		Time            time.Time `json:"time"`
		ReleaseTime     time.Time `json:"releaseTime"`
		Sha1            string    `json:"sha1"`
		ComplianceLevel int       `json:"complianceLevel"`
	} `json:"versions"`
}

var regexGameVersionId = regexp.MustCompile(`^[0-9]+\.[0-9]+(?:\.[0-9]+)?$`)

func (v *McVersions) gameVersions() ([]*semver.Version, error) {
	req, err := http.NewRequest(http.MethodGet, McVersionManifest, nil)
	if err != nil {
		return nil, err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var manifest PistonMetaManifest
	err = json.NewDecoder(resp.Body).Decode(&manifest)
	if err != nil {
		return nil, err
	}
	a := make([]*semver.Version, 0)
	for _, i := range manifest.Versions {
		if regexGameVersionId.MatchString(i.Id) {
			v, err := semver.NewVersion(i.Id)
			if err != nil {
				return nil, err
			}
			a = append(a, v)
		}
	}
	return a, err
}

func (v *McVersions) generateCache() {
	v.cacheMu.RLock()
	isValid := v.expires.Add(-2 * time.Hour).After(time.Now())
	v.cacheMu.RUnlock()
	if isValid {
		return
	}
	versions, err := v.gameVersions()
	if err != nil {
		log.Println("[MC Cache] Failed to fetch game versions:", err)
		return
	}
	slices.SortFunc(versions, func(a, b *semver.Version) int {
		return a.Compare(b)
	})
	v.versions = versions
}

func (v *McVersions) MatchingConstraints(c *semver.Constraints) []string {
	v.r.Run()
	v.r.Wait()
	v.cacheMu.RLock()
	defer v.cacheMu.RUnlock()
	a := make([]string, 0)
	for _, i := range v.versions {
		if c.Check(i) {
			a = append(a, toMcVersion(i))
		}
	}
	return a
}
