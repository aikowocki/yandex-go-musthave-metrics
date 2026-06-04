// Package resetexample содержит примеры структур для демонстрации
// работы кодогенератора cmd/reset.
package resetexample

// generate:reset
type Stats struct {
	Count   int
	Name    string
	Active  bool
	Values  []float64
	Labels  map[string]string
	Rate    float64
	NamePtr *string
	child   *Stats
}
