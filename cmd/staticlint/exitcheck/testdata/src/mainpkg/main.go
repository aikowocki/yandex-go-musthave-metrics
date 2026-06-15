package main

import "os"

func helper() {
	// Вне функции main вызов os.Exit допустим — диагностики быть не должно.
	os.Exit(1)
}

func main() {
	helper()
	os.Exit(0) // want "прямой вызов os.Exit запрещён в функции main"
}
