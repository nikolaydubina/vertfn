package vertfn

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

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
	reverse  bool
)

func init() {
	Analyzer.Flags.BoolVar(&verbose, "verbose", false, `print all details`)
	Analyzer.Flags.BoolVar(&colorize, "color", true, `colorize terminal`)
	Analyzer.Flags.BoolVar(&reverse, "reverse", false, `reverse ordering requirement`)
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

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

	check := func(ref *ast.Ident, def token.Pos, isRecvType bool) {
		if !def.IsValid() {
			// So far only seen on calls to Error method of error interface
			printer.Info(ref.Pos(), "got invalid definition position")
			return
		}
		// Ignore references to definitions that aren't at file scope, e.g. to local variables
		defFileScope := pass.Pkg.Scope().Innermost(pass.Fset.File(def).LineStart(1))
		defScope := pass.Pkg.Scope().Innermost(def)
		if defScope != defFileScope {
			printer.Info(ref.Pos(), fmt.Sprintf("skpping definition with inner scope %s != %s", defScope, defFileScope))
			return
		}

		kind := "definition"
		if isRecvType {
			kind = "receiver"
		}

		if pass.Fset.File(ref.Pos()).Name() != pass.Fset.File(def).Name() {
			printer.Ok(ref.Pos(), fmt.Sprintf(`%s %s in separate file (%s)`, kind, ref.Name, pass.Fset.Position(def)))
			return
		}

		refLine, defLine := pass.Fset.Position(ref.Pos()).Line, pass.Fset.Position(def).Line
		if refLine == defLine {
			printer.Ok(ref.Pos(), fmt.Sprintf(`%s %s used on same line as declared (%s)`, kind, ref.Name, pass.Fset.Position(def)))
			return
		}

		refBeforeDef := refLine < defLine
		order := "before"
		if !refBeforeDef {
			order = "after"
		}
		message := fmt.Sprintf(`%s %s used %s declared (%s)`, kind, ref.Name, order, pass.Fset.Position(def))

		// Written somewhat verbosely to help make it understandable
		orderOk := refBeforeDef
		if isRecvType || reverse {
			orderOk = !orderOk
		}
		if orderOk {
			printer.Ok(ref.Pos(), message)
		} else {
			printer.Error(ref.Pos(), message)
		}
	}

	// State for keeping track of the receiver type.
	// No need for a stack as method declarations can only be at file scope.
	var (
		funcDecl       *ast.FuncDecl
		recvType       *ast.TypeSpec
		beforeFuncType bool
	)

	inspect.Nodes(nil, func(n ast.Node, push bool) (proceed bool) {
		switch node := n.(type) {
		case *ast.File:
			if ast.IsGenerated(node) {
				printer.Info(node.Pos(), "skipping generated file")
				return false
			}

		case *ast.FuncDecl:
			if push && funcDecl == nil {
				funcDecl = node
				beforeFuncType = true
			} else if funcDecl == node {
				funcDecl = nil
			}

		case *ast.FuncType:
			beforeFuncType = false

		case *ast.SelectorExpr:
			sel := pass.TypesInfo.Selections[node]
			if sel == nil {
				// Based on TypesInfo.Selection docs this should only be the
				// case for "qualified identifiers", which I think means
				// references to out-of-package identifiers, which we don't care
				// about anyway. Logging just in case.
				printer.Info(node.Pos(), fmt.Sprintf("skipping selector %s with missing Selections", node.Sel.String()))
				break
			}

			obj := sel.Obj()
			switch sel.Kind() {
			case types.MethodVal:
				check(node.Sel, obj.Pos(), false)
			case types.FieldVal:
			case types.MethodExpr:
				check(node.Sel, obj.Pos(), false)
			default:
				// No other enum values are defined, logging just in case.
				printer.Info(node.Pos(), fmt.Sprintf("unknown selection kind %v", sel.Kind()))
			}

		case *ast.Ident:
			if node.Obj == nil {
				// Unclear when Obj is nil, but so far only cases where its ok:
				// import references, qualified identifiers, method names in
				// their definitions and package names in their definitions.
				printer.Info(node.Pos(), fmt.Sprintf("missing Obj for %s", node.Name))
				break
			}

			switch spec := node.Obj.Decl.(type) {
			case *ast.ValueSpec:
				for _, ident := range spec.Names {
					if ident.Name == node.Name && ident != node {
						check(node, ident.Pos(), false)
					}
				}
			case *ast.Field:
				// Explicitly log for easier debugging
				for _, ident := range spec.Names {
					if ident.Name == node.Name && ident != node {
						printer.Info(node.Pos(), fmt.Sprintf("skipping ident %s for field %s", node.Name, pass.Fset.Position(spec.Pos())))
					}
				}
			case *ast.FuncDecl:
				check(node, spec.Pos(), false)
			case *ast.TypeSpec:
				if funcDecl != nil && beforeFuncType {
					// We're in a file-level func decl before getting to the
					// function type, so this must be an identifier in the type
					// of the receiver.
					recvType = spec
					printer.Info(node.Pos(), fmt.Sprintf("skipping ident %s in recv list", node.Name))
					break
				}
				if funcDecl != nil && recvType == spec {
					// Reference to the receiver type within a method type or body
					check(node, spec.Pos(), true)
					break
				}
				check(node, spec.Pos(), false)
			default:
				printer.Info(node.Pos(), fmt.Sprintf("unexpected ident decl type %T", node.Obj.Decl))
			}
		}

		return true
	})

	return nil, nil
}
