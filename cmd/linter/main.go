// The linter command runs an analysis.
package main

import (
	"metralert/internal/linter"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() { singlechecker.Main(linter.Linter) }
