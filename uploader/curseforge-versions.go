package uploader

import (
	"slices"
	"strings"
)

func CfVersionId(versionTypes []CfVersionTypes, versions []CfVersions, ver string) (int, bool) {
	headerSlug := CfVersionHeaderSlug(ver)
	childSlug := CfVersionSlug(ver)
	n := slices.IndexFunc(versionTypes, func(a CfVersionTypes) bool { return a.Slug == headerSlug })
	if n == -1 {
		return 0, false
	}
	tId := versionTypes[n].Id
	n2 := slices.IndexFunc(versions, func(a CfVersions) bool { return a.GameVersionTypeID == tId && a.Slug == childSlug })
	if n2 == -1 {
		return 0, false
	}
	return versions[n2].Id, true
}

func CfVersionHeaderSlug(ver string) string {
	ver, _, _ = strings.Cut(ver, "-")
	a1, a2, found := strings.Cut(ver, ".")
	if !found {
		return ""
	}
	b1, _, found := strings.Cut(a2, ".")
	return "minecraft-" + a1 + "-" + b1
}

func CfVersionSlug(ver string) string {
	ver, _, _ = strings.Cut(ver, "-")
	return strings.ReplaceAll(ver, ".", "-")
}
