package packagesx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetPkgImportPathAndExpose(t *testing.T) {
	tt := require.New(t)

	cases := []struct {
		importPath string
		expose     string
		s          string
	}{
		{
			"",
			"B",
			"B",
		},
		{
			"testing",
			"B",
			"testing.B",
		},
		{
			"a.b.c.d/c",
			"B",
			"a.b.c.d/c.B",
		},
	}

	for _, caseItem := range cases {
		importPath, expose := GetPkgImportPathAndExpose(caseItem.s)
		tt.Equal(caseItem.importPath, importPath)
		tt.Equal(caseItem.expose, expose)
	}
}
