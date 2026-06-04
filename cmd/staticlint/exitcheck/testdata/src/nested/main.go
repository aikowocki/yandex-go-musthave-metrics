package main

import "os"

// Вызов os.Exit во вложенном замыкании внутри main также должен ловиться,
// потому что обход тела main рекурсивный.
func main() {
	func() {
		os.Exit(1) // want "прямой вызов os.Exit запрещён в функции main"
	}()
}
