package packagesx

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestGetPkgImportPathAndExpose(t *testing.T) {
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

		NewWithT(t).Expect(importPath).To(Equal(caseItem.importPath))
		NewWithT(t).Expect(expose).To(Equal(caseItem.expose))
	}
}
