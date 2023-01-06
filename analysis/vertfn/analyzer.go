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

func fnameFromCallExpr(n *ast.CallExpr) string {
	if ind, ok := n.Fun.(*ast.Ident); ok && ind != nil {
		return ind.Name
	}
	if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel != nil {
		// TODO: utilize type/package information
		// TODO: differentiate same method name but on different classes

		// TODO: chains of selectors
		// TODO: chains of methods
		fmt.Printf("%#v | %#v\n", sel.X, sel.Sel)
	}
	return ""
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
			fnCall[fnameFromCallExpr(call)] = append(fnCall[fnameFromCallExpr(call)], call)
		}
	})

	for fn, def := range fnDecl {
		fnDeclLineNum := pass.Fset.Position(def.Pos()).Line

		usedCount := 0
		for _, call := range fnCall[fn] {
			usedCount++
			fnCallLineNum := pass.Fset.Position(call.Pos()).Line

			if fnDeclLineNum > fnCallLineNum {
				printer.Ok(call.Pos(), fmt.Sprintf(`func %s delclared(%d) after used(%d)`, fn, fnDeclLineNum, fnCallLineNum))
			} else {
				printer.Error(call.Pos(), fmt.Sprintf(`func %s is declared(%d) before used(%d)`, fn, fnDeclLineNum, fnCallLineNum))
			}
		}

		if usedCount == 0 {
			printer.Info(def.Pos(), fmt.Sprintf(`func %s delclared(%d) but not used`, fn, fnDeclLineNum))
		}
	}

	return nil, nil
}
