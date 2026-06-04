package notmain

import "os"

// В пакете, отличном от main, прямой вызов os.Exit допустим даже в функции main.
func main() {
	os.Exit(1)
}
