package main

import (
	"CacheProject/internal/cache"
	"CacheProject/internal/config"
	"CacheProject/internal/handler"
	pb "CacheProject/proto"
	"context"
	"fmt"
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
	cfg := config.Load()

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
		Addr:    fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler: r,
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

	grpcServer.GracefulStop()
	log.Println("серверы остановлены")
}