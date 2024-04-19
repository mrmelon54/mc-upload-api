package jar_parser

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var forgeVersionRangeTestData = map[string]string{
	"1.0":             "=1.0",
	"1.0.0":           "=1.0.0",
	"[1.0,2.0)":       ">=1.0.0 <2.0.0",
	"[1.0,2.0]":       ">=1.0.0 <=2.0.0",
	"[1.0, 2.0)":      ">=1.0.0 <2.0.0",
	"[1.0, 2.0]":      ">=1.0.0 <=2.0.0",
	"[1.5,)":          ">=1.5.0",
	"(,1.0],[1.2,)":   "<=1.0.0 || >=1.2.0",
	"[1.16.4,1.16.5]": ">=1.16.4 <=1.16.5",
	"[1.16.4]":        "=1.16.4",
	"[1.20,1.20.1]":   ">=1.20.0 <=1.20.1",
}

func TestForgeVersionRange(t *testing.T) {
	for k, v := range forgeVersionRangeTestData {
		t.Run(k, func(t *testing.T) {
			a, err := ForgeVersionRange(k)
			assert.NoError(t, err)
			assert.Equal(t, v, a.String())
		})
	}
}
