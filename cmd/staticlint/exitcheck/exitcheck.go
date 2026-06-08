// Package exitcheck реализует статический анализатор, запрещающий прямые вызовы
// os.Exit в функции main пакета main. Это позволяет гарантировать корректный
// graceful shutdown через механизмы вроде signal.NotifyContext.
package exitcheck

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// Analyzer — анализатор, который находит вызовы os.Exit внутри func main() пакета main.
var Analyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "запрещает прямые вызовы os.Exit в функции main пакета main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Анализируем только package main — в остальных пакетах os.Exit допустим.
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		// Пропускаем сгенерированный код (например, _testmain.go от `go test`),
		// в котором os.Exit вставляется инструментами и допустим.
		if ast.IsGenerated(file) {
			continue
		}

		for _, decl := range file.Decls {
			// Ищем декларации функций.
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" {
				continue
			}

			// Обходим тело func main() рекурсивно.
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				// Ищем вызовы функций.
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}

				// Проверяем что вызов имеет вид pkg.Func (SelectorExpr).
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				// Вместо проверки имени — проверяем на какой объект ссылается sel.Sel
				obj := pass.TypesInfo.ObjectOf(sel.Sel)
				if obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == "os" && obj.Name() == "Exit" {
					pass.Reportf(call.Pos(), "прямой вызов os.Exit запрещён в функции main")
				}
				return true
			})
		}
	}
	return nil, nil
}
