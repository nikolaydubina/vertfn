package vertfn

import (
	"flag"
	"fmt"
	"go/ast"

	"github.com/nikolaydubina/vertfn/analysis/vertfn/color"

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
	verbose  bool
	colorize bool
)

func init() {
	Analyzer.Flags.BoolVar(&verbose, "verbose", false, `print all details`)
	Analyzer.Flags.BoolVar(&colorize, "color", true, `colorize terminal`)
}

func fnameFromFuncDecl(n *ast.FuncDecl) string { return n.Name.Name }

func fnameFromCallExpr(n *ast.CallExpr) []string {
	if ind, ok := n.Fun.(*ast.Ident); ok && ind != nil {
		return []string{ind.Name}
	}
	if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel != nil {
		var fnames []string
		fnames = append(fnames, sel.Sel.Name)
		if call, ok := sel.X.(*ast.CallExpr); ok && call != nil {
			fnames = append(fnames, fnameFromCallExpr(call)...)
		}
		return fnames
	}
	return nil
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	fnDecl := map[string]*ast.FuncDecl{}
	fnCall := map[string][]*ast.CallExpr{}

	var printer Printer = SimplePrinter{Pass: pass}
	if colorize {
		printer = ColorPrinter{
			Pass:       pass,
			ColorError: color.Red,
			ColorInfo:  color.Gray,
			ColorOk:    color.Green,
		}
	}
	printer = VerbosePrinter{Verbose: verbose, Printer: printer}
	printer = &SortedPrinter{Pass: pass, Printer: printer}
	defer printer.Flush()

	inspect.Preorder([]ast.Node{&ast.FuncDecl{}, &ast.CallExpr{}}, func(n ast.Node) {
		if fn, ok := n.(*ast.FuncDecl); ok && fn != nil {
			fnDecl[fnameFromFuncDecl(fn)] = fn
		}

		if call, ok := n.(*ast.CallExpr); ok && call != nil {
			for _, fname := range fnameFromCallExpr(call) {
				fnCall[fname] = append(fnCall[fname], call)
			}
		}
	})

	for fn, def := range fnDecl {
		fnDeclFile := pass.Fset.File(def.Pos()).Name()
		fnDeclLineNum := pass.Fset.Position(def.Pos()).Line

		usedCount := 0
		for _, call := range fnCall[fn] {
			usedCount++
			fnCallFile := pass.Fset.File(call.Pos()).Name()
			fnCallLineNum := pass.Fset.Position(call.Pos()).Line

			if fnCallFile != fnDeclFile {
				printer.Info(call.Pos(), fmt.Sprintf(`func %s declared in separate file(%s)`, fn, fnDeclFile))
				continue
			}

			if fnDeclLineNum > fnCallLineNum {
				printer.Ok(call.Pos(), fmt.Sprintf(`func %s declared(%d) after used(%d)`, fn, fnDeclLineNum, fnCallLineNum))
				continue
			}

			printer.Error(call.Pos(), fmt.Sprintf(`func %s is declared(%d) before used(%d)`, fn, fnDeclLineNum, fnCallLineNum))
		}

		if usedCount == 0 {
			printer.Info(def.Pos(), fmt.Sprintf(`func %s delclared(%d) but not used`, fn, fnDeclLineNum))
		}
	}

	return nil, nil
}
