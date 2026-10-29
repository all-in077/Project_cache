package handler

import (
	"context"
	"github.com/all-in077/Project_cache/internal/cache"
	pb "github.com/all-in077/Project_cache/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	pb.UnimplementedCacheServiceServer
	cache *cache.LRUcache
}

func NewGRPCHandler(cache *cache.LRUcache) *GRPCHandler {
	return &GRPCHandler{cache: cache}
}

func (g *GRPCHandler) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	value, found := g.cache.Get(req.Key)
	if !found {
		return &pb.GetResponse{Found: false}, nil
	}
	return &pb.GetResponse{Value: value, Found: true}, nil
}

func (g *GRPCHandler) Set(ctx context.Context, req *pb.SetRequest) (*pb.SetResponse, error) {
	if req.Key == "" || req.Value == nil {
		return nil, status.Error(codes.InvalidArgument, "key and value are required")
	}

	g.cache.Set(req.Key, req.Value)
	return &pb.SetResponse{}, nil
}

func (g *GRPCHandler) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	if req.Key == "" {
		return nil, status.Error(codes.InvalidArgument, "key is required")
	}
	ok := g.cache.Delete(req.Key)
	if !ok {
		return nil, status.Error(codes.NotFound, "key not found")
	}
	return &pb.DeleteResponse{Deleted: true}, nil
}

func (g *GRPCHandler) Stats(ctx context.Context, req *pb.StatsRequest) (*pb.StatsResponse, error) {
	s := g.cache.Stats()
	return &pb.StatsResponse{
		Hits:        s["hits"],
		Misses:      s["misses"],
		Evictions:   s["evictions"],
		CurrentSize: s["currentSize"],
	}, nil
}
