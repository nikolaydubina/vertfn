package vertfn

import (
	"go/token"
	"sort"

	"github.com/nikolaydubina/vertfn/analysis/vertfn/color"
	"golang.org/x/tools/go/analysis"
)

type Printer interface {
	Error(p token.Pos, s string)
	Info(p token.Pos, s string)
	Ok(p token.Pos, s string)
	Flush()
}

type SimplePrinter struct {
	Pass *analysis.Pass
}

func (c SimplePrinter) Error(p token.Pos, s string) { c.Pass.Reportf(p, s) }

func (c SimplePrinter) Info(p token.Pos, s string) { c.Pass.Reportf(p, s) }

func (c SimplePrinter) Ok(p token.Pos, s string) { c.Pass.Reportf(p, s) }

func (c SimplePrinter) Flush() {}

type VerbosePrinter struct {
	Verbose bool
	Printer Printer
}

func (c VerbosePrinter) Error(p token.Pos, s string) { c.Printer.Error(p, s) }

func (c VerbosePrinter) Info(p token.Pos, s string) {
	if c.Verbose {
		c.Printer.Info(p, s)
	}
}

func (c VerbosePrinter) Ok(p token.Pos, s string) {
	if c.Verbose {
		c.Printer.Ok(p, s)
	}
}

func (c VerbosePrinter) Flush() {}

type ColorPrinter struct {
	ColorError color.Color
	ColorInfo  color.Color
	ColorOk    color.Color
	Pass       *analysis.Pass
}

func (c ColorPrinter) Error(p token.Pos, s string) {
	c.Pass.Reportf(p, color.Colorize(c.ColorError, s))
}

func (c ColorPrinter) Info(p token.Pos, s string) {
	c.Pass.Reportf(p, color.Colorize(c.ColorInfo, s))
}

func (c ColorPrinter) Ok(p token.Pos, s string) {
	c.Pass.Reportf(p, color.Colorize(c.ColorOk, s))
}

func (c ColorPrinter) Flush() {}

// function call at position
type pcall struct {
	p token.Pos
	f func()
}

// SortedPrinter defers printin until Flush is called.
// Sorts print calls by line number of position.
type SortedPrinter struct {
	Printer Printer
	Pass    *analysis.Pass
	prints  []pcall
}

func (c *SortedPrinter) Flush() {
	sort.Slice(c.prints, func(i, j int) bool {
		iln := c.Pass.Fset.Position(c.prints[i].p).Line
		jln := c.Pass.Fset.Position(c.prints[j].p).Line
		return iln < jln
	})
	for _, pc := range c.prints {
		pc.f()
	}
}

func (c *SortedPrinter) Error(p token.Pos, s string) {
	c.prints = append(c.prints, pcall{p: p, f: func() { c.Printer.Error(p, s) }})
}

func (c *SortedPrinter) Info(p token.Pos, s string) {
	c.prints = append(c.prints, pcall{p: p, f: func() { c.Printer.Info(p, s) }})
}

func (c *SortedPrinter) Ok(p token.Pos, s string) {
	c.prints = append(c.prints, pcall{p: p, f: func() { c.Printer.Ok(p, s) }})
}
