package middleware

import (
	"fmt"
	"net/http"
	"slices"
)

func AllowMethods(methods ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("Method:", r.Method)
			if !slices.Contains(methods, r.Method) {
				http.Error(w, "Method not allowed2", http.StatusMethodNotAllowed)
				return
			}
			next(w, r)
		}
	}
}
