package main

import (
	"context"
	"fmt"
	"github.com/all-in077/Project_cache/internal/cache"
	"github.com/all-in077/Project_cache/internal/config"
	"github.com/all-in077/Project_cache/internal/handler"
	pb "github.com/all-in077/Project_cache/proto"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
)

func main() {
	// 1. Конфиг
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("конфиг: %v", err)
	}

	// 2. Контекст который отменится при сигнале остановки
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// 3. Кэш
	c := cache.NewLRUcache(cfg.CacheCapacity, cfg.CacheTTL, ctx)

	// 4. HTTP
	httpHandler := handler.NewHandler(c)
	r := chi.NewRouter()
	httpHandler.RegisterRoutes(r)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("HTTP сервер запущен на порту %s", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP сервер упал: %v", err)
		}
	}()

	// 5. gRPC
	grpcHandler := handler.NewGRPCHandler(c)
	grpcServer := grpc.NewServer()
	pb.RegisterCacheServiceServer(grpcServer, grpcHandler)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("не удалось занять порт: %v", err)
	}

	go func() {
		log.Printf("gRPC сервер запущен на порту %s", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC сервер упал: %v", err)
		}
	}()

	// 6. Ждём сигнала
	<-ctx.Done()
	log.Println("получен сигнал остановки")

	// 7. Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}

	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
	case <-shutdownCtx.Done():
		log.Println("gRPC graceful stop не уложился в таймаут, останавливаю принудительно")
		grpcServer.Stop()
	}
}
