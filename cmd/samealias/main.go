package main

import (
	"github.com/Rodge0/samealias"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(samealias.Analyzer)
}
