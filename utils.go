package packagesx

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"strings"
)

func ImportGoPath(importPath string) string {
	parts := strings.Split(importPath, "/vendor/")
	return parts[len(parts)-1]
}

func GetPkgImportPathAndExpose(s string) (string, string) {
	args := strings.Split(s, ".")
	lenOfArgs := len(args)
	if lenOfArgs > 1 {
		return ImportGoPath(strings.Join(args[0:lenOfArgs-1], ".")), args[lenOfArgs-1]
	}
	return "", s
}

func StringifyNode(fset *token.FileSet, node ast.Node) string {
	buf := bytes.Buffer{}
	if err := format.Node(&buf, fset, node); err != nil {
		panic(err)
	}
	return buf.String()
}

func GetIdentChainOfCallFunc(expr ast.Expr) (list []*ast.Ident) {
	switch e := expr.(type) {
	case *ast.CallExpr:
		list = append(list, GetIdentChainOfCallFunc(e.Fun)...)
	case *ast.SelectorExpr:
		list = append(list, GetIdentChainOfCallFunc(e.X)...)
		list = append(list, e.Sel)
	case *ast.Ident:
		list = append(list, expr.(*ast.Ident))
	}
	return
}

func Deref(tpe types.Type) types.Type {
	switch tpe.(type) {
	case *types.Pointer:
		return Deref(tpe.(*types.Pointer).Elem())
	}
	return tpe
}
