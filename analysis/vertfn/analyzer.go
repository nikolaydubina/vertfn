package vertfn

import (
	"flag"
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "vertfn",
	Doc:      "report vertical function ordering information",
	Run:      run,
	Flags:    flag.FlagSet{},
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

var (
	verbose bool
)

func init() {
	Analyzer.Flags.BoolVar(&verbose, "verbose", false, `print all details`)
}

func fnameFromFuncDecl(n *ast.FuncDecl) string { return n.Name.Name }

func fnameFromCallExpr(n *ast.CallExpr) string {
	if ind, ok := n.Fun.(*ast.Ident); ok && ind != nil {
		return ind.Name
	}
	if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel != nil {
		// TODO: utilize type/package information
		// TODO: chains of selectors
		// TODO: chains of methods
		// TODO: differentiate same method name but on different classes
		fmt.Printf("%#v | %#v\n", sel.X, sel.Sel)
	}
	return ""
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	fnDefLineNum := map[string]*ast.FuncDecl{}
	fnCallLineNum := map[string][]*ast.CallExpr{}

	inspect.Preorder([]ast.Node{&ast.FuncDecl{}, &ast.CallExpr{}}, func(n ast.Node) {
		if fn, ok := n.(*ast.FuncDecl); ok && fn != nil {
			fnDefLineNum[fnameFromFuncDecl(fn)] = fn
		}

		if call, ok := n.(*ast.CallExpr); ok && call != nil {
			fnCallLineNum[fnameFromCallExpr(call)] = append(fnCallLineNum[fnameFromCallExpr(call)], call)
		}
	})

	for fn, def := range fnDefLineNum {
		for _, call := range fnCallLineNum[fn] {
			if verbose || pass.Fset.Position(def.Pos()).Line > pass.Fset.Position(call.Pos()).Line {
				pass.Reportf(call.Pos(), `func %s is declared on line(%d) > used on line (%d)`, fn, 0, 0)
			}
		}
	}

	return nil, nil
}
