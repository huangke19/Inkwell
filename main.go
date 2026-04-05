package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"inkwell/config"
	"inkwell/db"
	"inkwell/handlers"
)

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		slog.Error("数据库初始化失败", "err", err)
		os.Exit(1)
	}
	defer database.Close()

	renderer := handlers.NewRenderer()
	wordHandler := handlers.NewWordHandler(database, renderer, cfg.GroqAPIKey)
	reviewHandler := handlers.NewReviewHandler(database, renderer, cfg.GroqAPIKey)

	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	mux.HandleFunc("GET /", wordHandler.List)
	mux.HandleFunc("GET /words/add", wordHandler.AddForm)
	mux.HandleFunc("POST /words", wordHandler.Create)
	mux.HandleFunc("GET /words/{id}", wordHandler.Detail)
	mux.HandleFunc("DELETE /words/{id}", wordHandler.Delete)
	mux.HandleFunc("GET /words/{id}/ai", wordHandler.GetAI)

	mux.HandleFunc("GET /review", reviewHandler.Start)
	mux.HandleFunc("POST /review/{id}", reviewHandler.Submit)
	mux.HandleFunc("GET /review/next", reviewHandler.Next)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 45 * time.Second,
	}

	go func() {
		slog.Info("启动服务", "addr", "http://localhost:"+cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("服务器错误", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("正在关闭服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
