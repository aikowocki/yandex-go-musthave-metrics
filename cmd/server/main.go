package main

import (
	"net/http"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/handler"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/middleware"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/repository"
)

func main() {
	storage := repository.NewMemStorage()
	updateHandler := handler.NewHandler(storage)

	// Создаем кастомный маршрутизатор
	mux := http.NewServeMux()
	// Доступен только POST метод
	mux.HandleFunc("/update/", middleware.AllowMethods(http.MethodPost)(updateHandler.Update))
	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
