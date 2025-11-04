package linter

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

var Linter = &analysis.Analyzer{
	Name: "panicexitchecker",
	Doc:  "check for panic, os.Exit and log.Fatal",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {

	for _, f := range pass.Files {

		ast.Inspect(f, func(n ast.Node) bool {
			// проверяем, какой конкретный тип лежит в узле
			if c, ok := n.(*ast.CallExpr); ok {
				switch fun := c.Fun.(type) {
				case *ast.Ident:
					if fun.Name == "panic" {
						pass.Reportf(fun.NamePos, "panic func is used")
					}
				case *ast.SelectorExpr:
					if f.Name.Name != "main" {
						if fun.Sel.Name == "Fatal" {
							if i, ok := fun.X.(*ast.Ident); i.Name == "log" && ok {
								pass.Reportf(fun.Sel.NamePos, "log.Fatal is used outside of main package")
							}
						}
						if fun.Sel.Name == "Exit" {
							if i, ok := fun.X.(*ast.Ident); i.Name == "os" && ok {
								pass.Reportf(fun.Sel.NamePos, "os.Exit is used outside of main package")
							}
						}
					}
				}
			}
			return true
		})
	}

	return nil, nil
}
