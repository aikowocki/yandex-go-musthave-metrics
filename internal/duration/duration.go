// Package duration содержит общие примитивы конфигурации, переиспользуемые
// конфигами агента и сервера.
package duration

import (
	"encoding/json"
	"strconv"
	"time"
)

// Seconds — обёртка над time.Duration с двумя форматами разбора:
//   - флаги и переменные окружения задают целое число секунд ("10");
//   - JSON задаёт строку длительности ("10s", "5m").
type Seconds time.Duration

// UnmarshalText разбирает целое число секунд (используется флагами и env).
func (s *Seconds) UnmarshalText(text []byte) error {
	v, err := strconv.Atoi(string(text))
	if err != nil {
		return err
	}
	*s = Seconds(time.Duration(v) * time.Second)
	return nil
}

// String возвращает значение в виде целого числа секунд.
func (s *Seconds) String() string {
	return strconv.Itoa(int(time.Duration(*s) / time.Second))
}

// Set реализует flag.Value через UnmarshalText.
func (s *Seconds) Set(val string) error {
	return s.UnmarshalText([]byte(val))
}

// UnmarshalJSON парсит строку длительности (например, "1s", "10s", "5m") из JSON.
func (s *Seconds) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	d, err := time.ParseDuration(str)
	if err != nil {
		return err
	}
	*s = Seconds(d)
	return nil
}
