package jar_parser

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

var fabricVersionRangeTestData = map[string]string{
	"[\"1.0\"]":                       "1.0",
	"\">=1.0.0\"":                     ">=1.0.0",
	"[\"1.0.0\",\"1.0.1\",\"1.0.2\"]": "1.0.0 || 1.0.1 || 1.0.2",
}

func TestFabricVersionRange(t *testing.T) {
	for k, v := range fabricVersionRangeTestData {
		t.Run(k, func(t *testing.T) {
			var f FabricVersionRange
			err := json.Unmarshal([]byte(k), &f)
			assert.NoError(t, err)
			assert.Equal(t, v, f.C.String())
		})
	}
}
