package jar_parser

import (
	"encoding/json"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"strings"
)

type FabricVersionRange struct {
	C *semver.Constraints
}

var _ json.Unmarshaler = &FabricVersionRange{}

func (v *FabricVersionRange) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return fmt.Errorf("invalid fabric version range")
	}
	switch b[0] {
	case '"':
		var s string
		err := json.Unmarshal(b, &s)
		if err != nil {
			return err
		}
		if len(s) == 0 {
			return fmt.Errorf("invalid fabric version range")
		}
		v.C, err = semver.NewConstraint(s)
		return err
	case '[':
		var s []string
		err := json.Unmarshal(b, &s)
		if err != nil {
			return err
		}
		v.C, err = semver.NewConstraint(strings.Join(s, " || "))
		return err
	}
	return fmt.Errorf("invalid fabric version range")
}
