package jar_parser

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

var forgeVersionRangeTestData = map[string]string{
	"1.0":           "=1.0",
	"1.0.0":         "=1.0.0",
	"[1.0,2.0)":     ">=1.0.0 <2.0.0",
	"[1.0,2.0]":     ">=1.0.0 <=2.0.0",
	"[1.5,)":        ">=1.5.0",
	"(,1.0],[1.2,)": "<=1.0.0 || >=1.2.0",
}

func TestForgeVersionRange(t *testing.T) {
	for k, v := range forgeVersionRangeTestData {
		t.Run(k, func(t *testing.T) {
			a, err := ForgeVersionRange(k)
			assert.NoError(t, err)
			fmt.Println(v)
			assert.Equal(t, v, a.String())
		})
	}
}
