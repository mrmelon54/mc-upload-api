package jar_parser

import (
	"errors"
	"github.com/Masterminds/semver/v3"
	"log"
	"regexp"
	"strings"
)

var regexVersionRange = regexp.MustCompile(`^(?:([(\[])([0-9]+(?:\.[0-9]+(?:\.[0-9]+)?)?)?(, ?([0-9]+(?:\.[0-9]+(?:\.[0-9]+)?)?)?)?([)\]])|([0-9]+(?:\.[0-9]+(?:\.[0-9]+)?)?)?)`)

type forgeVersionEnd struct {
	V        *semver.Version
	included bool
}

func parseForgeVersionEnd(s string, included bool) (*forgeVersionEnd, error) {
	if s == "" {
		return &forgeVersionEnd{nil, included}, nil
	}

	version, err := semver.NewVersion(s)
	log.Println(s, version, err)
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

		log.Printf("%#v\n", submatch)
		log.Printf("%#v - %#v\n", submatch[2], submatch[4])
		if submatch[1] == "" {
			return semver.NewConstraint("=" + submatch[6])
		}
		start, err = parseForgeVersionEnd(submatch[2], submatch[1] == "[")
		if err != nil {
			return nil, err
		}
		end, err = parseForgeVersionEnd(submatch[4], submatch[5] == "]")
		if err != nil {
			return nil, err
		}

		// detect single item in range [1.16.5]
		if submatch[4] == "" && !strings.HasPrefix(submatch[3], ",") {
			log.Printf("%#v - %#v - %#v\n", submatch, start, end)
			if !start.included || !end.included {
				return nil, ErrInvalidForgeVersionRange
			}
			return semver.NewConstraint("=" + submatch[2])
		}

		if start.V != nil {
			sb.WriteByte('>')
			if start.included {
				sb.WriteByte('=')
			}
			sb.WriteString(start.V.String())
			if end.V != nil {
				sb.WriteString(",")
			}
		}
		if end.V != nil {
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
