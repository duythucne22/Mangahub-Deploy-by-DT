package grpc

import (
	"context"
	"fmt"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "mangahub/internal/protocols/grpc/pb"
	"mangahub/pkg/models"
)

// Client wraps gRPC manga service client
type Client struct {
	conn   *grpc.ClientConn
	client pb.MangaServiceClient
}

// NewClient creates a new gRPC client
func NewClient(addr string) (*Client, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	return &Client{
		conn:   conn,
		client: pb.NewMangaServiceClient(conn),
	}, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// StreamSearch performs streaming search as user types
func (c *Client) StreamSearch(ctx context.Context, query string, results chan<- models.Manga, done chan<- error) {
	defer close(results)
	defer close(done)

	req := &pb.SearchRequest{
		Query:  query,
		Limit:  20,
		Offset: 0,
	}

	stream, err := c.client.StreamSearch(ctx, req)
	if err != nil {
		done <- fmt.Errorf("failed to start stream: %w", err)
		return
	}

	for {
		manga, err := stream.Recv()
		if err == io.EOF {
			done <- nil
			return
		}
		if err != nil {
			done <- fmt.Errorf("stream error: %w", err)
			return
		}

		// Convert protobuf manga to models.Manga
		results <- models.Manga{
			ID:          manga.Id,
			Title:       manga.Title,
			Description: manga.Description,
			CoverURL:    manga.CoverUrl,
			Status:      manga.Status,
		}
	}
}

// StreamSearchResults collects streaming search results into a slice
func (c *Client) StreamSearchResults(ctx context.Context, query string) ([]models.Manga, error) {
	resultsCh := make(chan models.Manga)
	doneCh := make(chan error, 1)

	go c.StreamSearch(ctx, query, resultsCh, doneCh)

	var results []models.Manga
	for {
		select {
		case manga, ok := <-resultsCh:
			if ok {
				results = append(results, manga)
			} else {
				resultsCh = nil
			}
		case err, ok := <-doneCh:
			if ok && err != nil {
				return nil, err
			}
			return results, nil
		}
	}
}

// Search performs a standard search (fallback if streaming not available)
func (c *Client) Search(ctx context.Context, query string, limit, offset int) ([]models.Manga, int, error) {
	req := &pb.SearchRequest{
		Query:  query,
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	resp, err := c.client.SearchManga(ctx, req)
	if err != nil {
		return nil, 0, fmt.Errorf("search failed: %w", err)
	}

	manga := make([]models.Manga, len(resp.Manga))
	for i, m := range resp.Manga {
		manga[i] = models.Manga{
			ID:          m.Id,
			Title:       m.Title,
			Description: m.Description,
			CoverURL:    m.CoverUrl,
			Status:      m.Status,
		}
	}

	return manga, int(resp.Total), nil
}
