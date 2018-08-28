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
	"github.com/stretchr/testify/require"
)

func TestPackage(t *testing.T) {
	cwd, _ := os.Getwd()
	pkg, err := Load(filepath.Join(cwd, "./__fixtures__"))
	require.NoError(t, err)
	require.NotEmpty(t, pkg.AllPackages)

	{
		tpeName := pkg.TypeName("Date")
		require.Equal(t, "type Date", pkg.CommentsOf(pkg.IdentOf(tpeName)))
	}

	{
		tpeName := pkg.Var("test")
		require.Equal(t, "var", pkg.CommentsOf(pkg.IdentOf(tpeName)))
	}

	{
		tpeName := pkg.Const("A")
		require.Equal(t, "a\n\nA", pkg.CommentsOf(pkg.IdentOf(tpeName)))
	}

	{
		tpeName := pkg.Func("Print")
		require.Equal(t, "func Print", pkg.CommentsOf(pkg.IdentOf(tpeName)))
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
			require.Len(t, values, n)
			require.Equal(t, c.results, printValues(pkg.Fset, values))
		})
	}

	{
		method, _ := typesutil.FromTType(pkg.TypeName("String").Type()).MethodByName("Method")
		values, n := pkg.FuncResultsOf(method.(*typesutil.TMethod).Func)
		require.Equal(t, 1, n)
		require.Len(t, values, n)
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
