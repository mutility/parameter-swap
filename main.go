// Package main hosts the parameter-swap analyzer pswap.Analyzer()
package main

import (
	"github.com/mutility/parameter-swap/pswap"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(pswap.Analyzer().Analyzer)
}
