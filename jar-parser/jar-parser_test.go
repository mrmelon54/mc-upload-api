package jar_parser

import (
	"bytes"
	"embed"
	"github.com/stretchr/testify/assert"
	"testing"
)

//go:embed test-*.jar
var testJars embed.FS

var platforms = []string{"fabric", "forge", "neoforge", "quilt"}

func TestJarParser(t *testing.T) {
	for _, i := range platforms {
		testJarBytes, err := testJars.ReadFile("test-" + i + ".jar")
		assert.NoError(t, err)
		t.Run(i, func(t *testing.T) {
			metadata, err := JarParser(bytes.NewReader(testJarBytes), int64(len(testJarBytes)))
			assert.NoError(t, err)
			assert.Equal(t, []string{i}, metadata.Loaders)
		})
	}
}
