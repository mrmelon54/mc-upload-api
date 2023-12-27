package jar_parser

import (
	"errors"
	"github.com/Masterminds/semver/v3"
	"regexp"
	"strings"
)

var regexVersionRange = regexp.MustCompile(`^([(\[])([0-9]+(?:\.[0-9]+(?:\.[0-9]+)?)?)?,([0-9]+(?:\.[0-9]+(?:\.[0-9]+)?)?)?([)\]])`)

type forgeVersionEnd struct {
	V        *semver.Version
	included bool
}

func parseForgeVersionEnd(s string, included bool) (*forgeVersionEnd, error) {
	if s == "" {
		return nil, nil
	}
	version, err := semver.NewVersion(s)
	if err != nil {
		return nil, err
	}
	return &forgeVersionEnd{version, included}, nil
}

var ErrInvalidForgeVersionRange = errors.New("invalid forge version range")

func ForgeVersionRange(s string) (constraints *semver.Constraints, err error) {
	var sb strings.Builder
	for len(s) > 0 {
		if sb.Len() > 0 {
			sb.WriteString(" || ")
		}

		var start, end *forgeVersionEnd
		submatch := regexVersionRange.FindStringSubmatch(s)
		if submatch == nil {
			return nil, ErrInvalidForgeVersionRange
		}
		start, err = parseForgeVersionEnd(submatch[2], submatch[1] == "[")
		if err != nil {
			return nil, err
		}
		end, err = parseForgeVersionEnd(submatch[3], submatch[4] == "]")
		if err != nil {
			return nil, err
		}

		if start != nil {
			sb.WriteByte('>')
			if start.included {
				sb.WriteByte('=')
			}
			sb.WriteString(start.V.String())
			if end != nil {
				sb.WriteString(",")
			}
		}
		if end != nil {
			sb.WriteByte('<')
			if end.included {
				sb.WriteByte('=')
			}
			sb.WriteString(end.V.String())
		}

		s = s[len(submatch[0]):]
		if len(s) > 0 {
			if s[0] != ',' {
				return nil, ErrInvalidForgeVersionRange
			}
			s = s[1:]
		}
	}
	return semver.NewConstraint(sb.String())
}
