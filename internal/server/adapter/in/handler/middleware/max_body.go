package middleware

import "net/http"

// DefaultMaxBodyBytes ограничивает размер сырого тела входящего запроса.
const DefaultMaxBodyBytes int64 = 1 << 20

// WithMaxBodySize middleware следует ставить максимально рано в цепочке — до gzip и hash,
// чтобы лимит применялся к сырому потоку.
func WithMaxBodySize(max int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, max)
			}
			next.ServeHTTP(w, r)
		})
	}
}
