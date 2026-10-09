package services

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "backend/internal/gen/analytics"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AnalyticsClient wraps the generated gRPC client stub
type AnalyticsClient struct {
	client pb.AnalyticsServiceClient
	conn   *grpc.ClientConn
}

// NewAnalyticsClient establishes a persistent HTTP/2 gRPC connection to the Analytics service
func NewAnalyticsClient(targetAddr string) (*AnalyticsClient, error) {
	// grpc.WithTransportCredentials(insecure.NewCredentials()) is used because
	// communication is within the private internal Docker network (no TLS needed)
	conn, err := grpc.NewClient(
		targetAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to analytics gRPC: %w", err)
	}

	client := pb.NewAnalyticsServiceClient(conn)
	log.Printf("Connected to Analytics gRPC service at %s\n", targetAddr)

	return &AnalyticsClient{
		client: client,
		conn:   conn,
	}, nil
}

// Close gracefully closes the underlying TCP connection when the backend shuts down
func (a *AnalyticsClient) Close() error {
	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}

// GetPackEV sends a batch of card models to Python and receives calculated EV metrics
func (a *AnalyticsClient) GetPackEV(ctx context.Context, setCode string, cards []*pb.CardData) (*pb.EVResponse, error) {
	// Best practice: Always set an RPC timeout (deadline) so slow analytics never hang the Go server
	rpcCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &pb.EVRequest{
		SetCode: setCode,
		Cards:   cards,
	}

	resp, err := a.client.CalculatePackEV(rpcCtx, req)
	if err != nil {
		return nil, fmt.Errorf("analytics gRPC CalculatePackEV failed: %w", err)
	}

	return resp, nil
}
