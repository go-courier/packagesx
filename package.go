package packagesx

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

type Poser interface {
	Pos() token.Pos
}

func Load(pattern string) (*Package, error) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.LoadAllSyntax,
	}, pattern)

	if err != nil {
		return nil, err
	}

	return NewPackage(pkgs[0]), nil
}

func NewPackage(pkg *packages.Package) *Package {
	p := &Package{
		Package: pkg,
	}

	s := pkgSet{}
	s.add(pkg)

	p.AllPackages = s.allPackages()

	return p
}

type pkgSet map[string]*packages.Package

func (s pkgSet) add(pkg *packages.Package) {
	s[pkg.ID] = pkg

	for k := range pkg.Imports {
		s.add(pkg.Imports[k])
	}
}

func (s pkgSet) allPackages() []*packages.Package {
	list := make([]*packages.Package, 0)
	for id := range s {
		list = append(list, s[id])
	}
	return list
}

type Package struct {
	*packages.Package
	AllPackages []*packages.Package
}

func (p *Package) Const(name string) *types.Const {
	for ident, def := range p.TypesInfo.Defs {
		if typeConst, ok := def.(*types.Const); ok {
			if ident.Name == name {
				return typeConst
			}
		}
	}
	return nil
}

func (p *Package) TypeName(name string) *types.TypeName {
	for ident, def := range p.TypesInfo.Defs {
		if typeName, ok := def.(*types.TypeName); ok {
			if ident.Name == name {
				return typeName
			}
		}
	}
	return nil
}

func (p *Package) Var(name string) *types.Var {
	for ident, def := range p.TypesInfo.Defs {
		if typeVar, ok := def.(*types.Var); ok {
			if ident.Name == name {
				return typeVar
			}
		}
	}
	return nil
}

func (p *Package) Func(name string) *types.Func {
	for ident, def := range p.TypesInfo.Defs {
		if typeFunc, ok := def.(*types.Func); ok {
			if ident.Name == name {
				return typeFunc
			}
		}
	}
	return nil
}

func (prog *Package) Pkg(importPath string) *packages.Package {
	for _, pkg := range prog.AllPackages {
		pkgPath := pkg.PkgPath
		if importPath == pkgPath {
			return pkg
		}
	}
	return nil
}

func (prog *Package) PkgOf(poser Poser) *types.Package {
	for _, pkg := range prog.AllPackages {
		for _, file := range pkg.Syntax {
			if file.Pos() <= poser.Pos() && file.End() > poser.Pos() {
				return pkg.Types
			}
		}
	}
	return nil
}

func (prog *Package) PkgInfoOf(poser Poser) *types.Info {
	for _, pkg := range prog.AllPackages {
		for _, file := range pkg.Syntax {
			if file.Pos() <= poser.Pos() && file.End() > poser.Pos() {
				return pkg.TypesInfo
			}
		}
	}
	return nil
}

func (prog *Package) FileOf(poser Poser) *ast.File {
	for _, pkg := range prog.AllPackages {
		for _, file := range pkg.Syntax {
			if file.Pos() <= poser.Pos() && file.End() > poser.Pos() {
				return file
			}
		}
	}
	return nil
}

func (prog *Package) IdentOf(obj types.Object) *ast.Ident {
	pkgInfo := prog.Pkg(obj.Pkg().Path())

	for ident, def := range pkgInfo.TypesInfo.Defs {
		if def == obj {
			return ident
		}
	}
	return nil
}

func (prog *Package) CommentsOf(node ast.Node) string {
	file := prog.FileOf(node)
	if file == nil {
		return ""
	}
	commentScanner := NewCommentScanner(prog.Fset, file)
	doc := commentScanner.CommentsOf(node)
	if doc != "" {
		return doc
	}
	return doc
}

func (prog *Package) Eval(expr ast.Expr) (types.TypeAndValue, error) {
	return types.Eval(prog.Fset, prog.PkgOf(expr), expr.Pos(), StringifyNode(prog.Fset, expr))
}

func (prog *Package) FuncDeclOf(typeFunc *types.Func) (funcDecl *ast.FuncDecl) {
	ast.Inspect(prog.FileOf(typeFunc), func(node ast.Node) bool {
		if decl, ok := node.(*ast.FuncDecl); ok {
			if decl.Pos() <= typeFunc.Pos() && decl.Body != nil && typeFunc.Pos() < decl.Body.Pos() {
				funcDecl = decl
				return false
			}
		}
		return true
	})
	return
}

type Results map[int][]TypeAndValueWithExpr

func blockContainsReturn(node ast.Node) (ok bool) {
	ast.Inspect(node, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.ReturnStmt:
			ok = true
		}
		return true
	})
	return
}

func (prog *Package) funcResultsOfCallExpr(callExpr *ast.CallExpr) (Results, int) {
	typ := prog.PkgInfoOf(callExpr).TypeOf(callExpr)
	results := Results{}

	switch typ := typ.(type) {
	case *types.Tuple:
		for i := 0; i < typ.Len(); i++ {
			prog.appendResult(results, i, TypeAndValueWithExpr{
				TypeAndValue: types.TypeAndValue{
					Type: typ.At(i).Type(),
				},
				Expr: callExpr,
			})
		}
	default:
		prog.appendResult(results, 0, TypeAndValueWithExpr{
			TypeAndValue: types.TypeAndValue{
				Type: typ,
			},
			Expr: callExpr,
		})
	}
	return results, len(results)
}

func (prog *Package) getAssignedValueOf(ident *ast.Ident, returnPos token.Pos) []TypeAndValueWithExpr {
	assignStmt := (*ast.AssignStmt)(nil)
	idx := 0

	file := prog.FileOf(ident)

	block := (*ast.BlockStmt)(nil)

	ast.Inspect(file, func(node ast.Node) bool {
		switch fn := node.(type) {
		case *ast.FuncLit:
			if fn.Pos() <= ident.Pos() && ident.Pos() <= fn.End() {
				block = fn.Body
			}
			return false
		case *ast.FuncDecl:
			if fn.Pos() <= ident.Pos() && ident.Pos() <= fn.End() {
				block = fn.Body
			}
			return false
		}
		return true
	})

	if block == nil {
		return nil
	}

	// scan latest AssignStmt before return
	scan := func(astNode ast.Node) {
		queue := []ast.Node{astNode}
		for len(queue) > 0 {
			astNode = queue[0]
			queue = queue[1:]

			ast.Inspect(astNode, func(node ast.Node) bool {
				if node == nil || node.Pos() >= returnPos {
					return false
				}
				switch stmt := node.(type) {
				case *ast.CaseClause:
					return !blockContainsReturn(stmt) || stmt.Pos() <= returnPos && returnPos < stmt.End()
				case *ast.IfStmt:
					if stmt.Else != nil {
						queue = append(queue, stmt.Else)
					}
					return !blockContainsReturn(stmt.Body) || stmt.Body.Pos() <= returnPos && returnPos < stmt.Body.End()
				case *ast.AssignStmt:
					for i := range stmt.Lhs {
						if id, ok := stmt.Lhs[i].(*ast.Ident); ok {
							if ident.Obj == id.Obj {
								assignStmt = stmt
								idx = i
							}
						}

					}
				}
				return true
			})
		}
	}

	scan(block)

	if assignStmt == nil {
		return nil
	}

	results := Results{}
	prog.setResultsByExprList(results, assignStmt.Rhs...)
	return results[idx]

}

func (prog *Package) appendResult(results Results, i int, typeAndValue TypeAndValueWithExpr) {
	if _, ok := typeAndValue.Type.(*types.Interface); ok {
		switch e := typeAndValue.Expr.(type) {
		case *ast.Ident:
			results[i] = append(results[i], prog.getAssignedValueOf(e, e.Pos())...)
		case *ast.SelectorExpr:
			results[i] = append(results[i], prog.getAssignedValueOf(e.Sel, e.Sel.Pos())...)
		default:
			results[i] = append(results[i], typeAndValue)
		}
		return
	}
	results[i] = append(results[i], typeAndValue)
}

func (prog *Package) setResultsByExprList(results Results, exprs ...ast.Expr) {
	for i := range exprs {
		switch e := exprs[i].(type) {
		case *ast.CallExpr:
			callResults, callResultsLength := prog.funcResultsOfCallExpr(e)
			for j := 0; j < callResultsLength; j++ {
				if j > 0 {
					i++
				}
				for _, tv := range callResults[j] {
					results[i] = append(results[i], TypeAndValueWithExpr{
						TypeAndValue: tv.TypeAndValue,
						Expr:         e,
					})
				}
			}
		default:
			tv, _ := prog.Eval(e)
			prog.appendResult(results, i, TypeAndValueWithExpr{
				TypeAndValue: tv,
				Expr:         e,
			})
		}
	}
}

func (prog *Package) FuncResultsOf(typeFunc *types.Func) (Results, int) {
	if typeFunc == nil {
		return nil, 0
	}

	funcDecl := prog.FuncDeclOf(typeFunc)
	if funcDecl == nil {
		return nil, 0
	}
	// TODO find way to location interface

	return prog.FuncResultsOfSignature(typeFunc.Type().(*types.Signature), funcDecl.Body, funcDecl.Type)
}

func (prog *Package) FuncResultsOfSignature(signature *types.Signature, funcBody *ast.BlockStmt, astFuncType *ast.FuncType) (Results, int) {
	resultTypes := signature.Results()
	if resultTypes.Len() == 0 {
		return nil, 0
	}

	namedResults := make([]*ast.Ident, 0)

	for _, field := range astFuncType.Results.List {
		for _, name := range field.Names {
			namedResults = append(namedResults, name)
		}
	}

	// collect all return stmt
	getReturnStmtList := func() []*ast.ReturnStmt {
		returnStmtList := make([]*ast.ReturnStmt, 0)

		ast.Inspect(funcBody, func(node ast.Node) bool {
			switch node.(type) {
			case *ast.FuncLit:
				return false // skip func inline declaration
			case *ast.ReturnStmt:
				returnStmtList = append(returnStmtList, node.(*ast.ReturnStmt))
			}
			return true
		})

		return returnStmtList
	}

	finalReturns := Results{}

	for _, returnStmt := range getReturnStmtList() {
		if returnStmt.Results == nil {
			for i := 0; i < resultTypes.Len(); i++ {
				// named returns
				for _, tv := range prog.getAssignedValueOf(namedResults[i], returnStmt.Pos()) {
					prog.appendResult(finalReturns, i, tv)
				}
			}
			continue
		}
		prog.setResultsByExprList(finalReturns, returnStmt.Results...)
	}

	for i := range finalReturns {
		for j := range finalReturns[i] {
			tve := finalReturns[i][j]

			// patch type of typeAndValue
			// to convert value to the matched result type
			switch tpe := resultTypes.At(i).Type().(type) {
			case *types.Interface:
				// do nothing
			case *types.Named:
				if tpe.String() != "error" {
					tve.Type = tpe
				}
			default:
				tve.Type = tpe
			}

			finalReturns[i][j] = tve
		}
	}

	return finalReturns, resultTypes.Len()
}

type TypeAndValueWithExpr struct {
	Expr ast.Expr
	types.TypeAndValue
}
