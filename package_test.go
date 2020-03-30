package packagesx

import (
	"bytes"
	"fmt"
	"go/format"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-courier/reflectx/typesutil"
	. "github.com/onsi/gomega"
)

func TestPackage(t *testing.T) {
	cwd, _ := os.Getwd()
	pkg, err := Load(filepath.Join(cwd, "./__fixtures__"))

	NewWithT(t).Expect(err).To(BeNil())
	NewWithT(t).Expect(pkg.AllPackages).NotTo(BeEmpty())

	{
		tpeName := pkg.TypeName("Date")
		NewWithT(t).Expect(pkg.CommentsOf(pkg.IdentOf(tpeName))).To(Equal("type Date"))
	}

	{
		tpeName := pkg.Var("test")
		NewWithT(t).Expect(pkg.CommentsOf(pkg.IdentOf(tpeName))).To(Equal("var"))
	}

	{
		tpeName := pkg.Const("A")
		NewWithT(t).Expect(pkg.CommentsOf(pkg.IdentOf(tpeName))).To(Equal("a\n\nA"))
	}

	{
		tpeName := pkg.Func("Print")
		NewWithT(t).Expect(pkg.CommentsOf(pkg.IdentOf(tpeName))).To(Equal("func Print"))
	}

	cases := []struct {
		funcName string
		results  [][]string
	}{
		{
			"FuncCallReturnAssign",
			[][]string{
				{"interface{}"},
				{"github.com/go-courier/packagesx/__fixtures__.String"},
			},
		},
		{
			"FuncCallWithFuncLit",
			[][]string{
				{"interface{}"},
				{"github.com/go-courier/packagesx/__fixtures__.String(\"1\")"},
			},
		},
		{
			"FuncSingleReturn",
			[][]string{
				{"untyped int(2)"},
			},
		},
		{
			"FuncSelectExprReturn",
			[][]string{
				{"string"},
			},
		},
		{
			"FuncWillCall",
			[][]string{
				{"interface{}"},
				{`github.com/go-courier/packagesx/__fixtures__.String`},
			},
		},
		{
			"FuncReturnWithCallDirectly",
			[][]string{
				{"interface{}"},
				{`github.com/go-courier/packagesx/__fixtures__.String`},
			},
		},
		{
			"FuncWithNamedReturn",
			[][]string{
				{"interface{}"},
				{`github.com/go-courier/packagesx/__fixtures__.String`},
			},
		},
		{
			"FuncSingleNamedReturnByAssign",
			[][]string{
				{`untyped string("1")`},
				{`github.com/go-courier/packagesx/__fixtures__.String("2")`},
			},
		},
		{
			"FunWithSwitch",
			[][]string{
				{`untyped string("a1")`, `untyped string("a2")`, `untyped string("a3")`},
				{
					`github.com/go-courier/packagesx/__fixtures__.String("b1")`,
					`github.com/go-courier/packagesx/__fixtures__.String("b2")`,
					`github.com/go-courier/packagesx/__fixtures__.String("b3")`,
				},
			},
		},
		{
			"FuncWithIf",
			[][]string{
				{`untyped string("a0")`, `untyped string("a1")`, `string`},
			},
		},
		{
			"FuncWithCurryCall",
			[][]string{
				{`int`},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.funcName, func(t *testing.T) {
			values, n := pkg.FuncResultsOf(pkg.Func(c.funcName))
			NewWithT(t).Expect(values).To(HaveLen(n))
			NewWithT(t).Expect(c.results).To(Equal(printValues(pkg.Fset, values)))
		})
	}

	{
		method, _ := typesutil.FromTType(pkg.TypeName("String").Type()).MethodByName("Method")
		values, n := pkg.FuncResultsOf(method.(*typesutil.TMethod).Func)

		NewWithT(t).Expect(n).To(Equal(1))
		NewWithT(t).Expect(values).To(HaveLen(n))
	}
}

func printValues(fset *token.FileSet, results map[int][]TypeAndValueWithExpr) [][]string {
	if results == nil {
		return [][]string{}
	}

	s := make([][]string, len(results))

	for i := range s {
		tvs := results[i]
		s[i] = make([]string, len(tvs))
		for j, tv := range tvs {
			buf := bytes.NewBuffer(nil)
			format.Node(buf, fset, tv.Expr)
			fmt.Println(buf.String())

			if tv.Value == nil {
				s[i][j] = fmt.Sprintf("%s", tv.Type.String())
				continue
			}
			s[i][j] = fmt.Sprintf("%s(%s)", tv.Type.String(), tv.Value)
		}
	}

	return s
}
