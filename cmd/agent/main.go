package main

import (
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/agent"
)

func main() {
	storage := agent.NewLocalStorage()
	client := agent.NewClient("http://localhost:8080")

	i := 0
	for {
		agent.CollectMetrics(storage)
		i++
		if i%5 == 0 {
			agent.Report(storage, client)
		}
		time.Sleep(2 * time.Second)
	}
}
