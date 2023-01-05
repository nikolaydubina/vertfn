package main

import (
	"github.com/nikolaydubina/vertfn/analysis/vertfn"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() { singlechecker.Main(vertfn.Analyzer) }
