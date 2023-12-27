package resolve_versions

import (
	"github.com/Masterminds/semver/v3"
	mapset "github.com/deckarep/golang-set/v2"
)

func ResolveGameVersions(constraints []*semver.Constraints, mcVersions *McVersions) ([]string, error) {
	verSet := mapset.NewThreadUnsafeSet[string]()
	for _, constraint := range constraints {
		a := mcVersions.MatchingConstraints(constraint)
		verSet.Append(a...)
	}
	return verSet.ToSlice(), nil
}
