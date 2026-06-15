package main

import osalias "os"

// Импорт os под алиасом: детект по имени пакета "os" в тексте сломался бы,
// но проверка через TypesInfo по-прежнему находит вызов.
func main() {
	osalias.Exit(1) // want "прямой вызов os.Exit запрещён в функции main"
}
