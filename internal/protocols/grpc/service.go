package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "mangahub/internal/protocols/grpc/pb"
	"mangahub/internal/repository"
	"mangahub/pkg/models"
)

// MangaServiceServer implements the gRPC MangaService
type MangaServiceServer struct {
	pb.UnimplementedMangaServiceServer
	pool      *pgxpool.Pool
	mangaRepo repository.MangaRepository
	statsRepo repository.StatsRepository
}

// NewMangaServiceServer creates a new gRPC manga service
func NewMangaServiceServer(pool *pgxpool.Pool, mangaRepo repository.MangaRepository, statsRepo repository.StatsRepository) *MangaServiceServer {
	return &MangaServiceServer{
		pool:      pool,
		mangaRepo: mangaRepo,
		statsRepo: statsRepo,
	}
}

// StreamSearch streams manga search results in real-time
func (s *MangaServiceServer) StreamSearch(req *pb.SearchRequest, stream pb.MangaService_StreamSearchServer) error {
	logger := logrus.StandardLogger()
	ctx := stream.Context()

	// Validate request
	if err := s.validateSearchRequest(req); err != nil {
		return err
	}

	// Start with empty query to show recent manga
	if strings.TrimSpace(req.Query) == "" {
		return s.streamRecentManga(ctx, stream, req.Limit)
	}

	// Process query character by character for true streaming
	searchTerm := strings.ToLower(strings.TrimSpace(req.Query))
	if len(searchTerm) < 2 {
		return s.streamRecentManga(ctx, stream, req.Limit)
	}

	useShortMatch := len(searchTerm) < 3

	var baseQuery string
	if useShortMatch {
		baseQuery = `
	SELECT 
		m.id, m.title, m.description, m.cover_url, m.status,
		0 as relevance_score,
		COALESCE(s.comment_count, 0) as comment_count,
		COALESCE(s.chat_count, 0) as chat_count,
		COALESCE(s.weekly_score, 0) as weekly_score,
		m.created_at,
		COALESCE((
			SELECT json_agg(json_build_object('id', g.id, 'name', g.name) ORDER BY g.name)
			FROM manga_genres mg
			INNER JOIN genres g ON mg.genre_id = g.id
			WHERE mg.manga_id = m.id
		), '[]'::json) as genres
	FROM manga m
	LEFT JOIN manga_stats s ON m.id = s.manga_id
	WHERE LOWER(m.title) LIKE '%' || $1 || '%'
	`
	} else {
		// Build base query with FTS
		baseQuery = `
	SELECT 
		m.id, m.title, m.description, m.cover_url, m.status,
		ts_rank_cd(m.search_vector, websearch_to_tsquery('english', $1)) as relevance_score,
		COALESCE(s.comment_count, 0) as comment_count,
		COALESCE(s.chat_count, 0) as chat_count,
		COALESCE(s.weekly_score, 0) as weekly_score,
		m.created_at,
		COALESCE((
			SELECT json_agg(json_build_object('id', g.id, 'name', g.name) ORDER BY g.name)
			FROM manga_genres mg
			INNER JOIN genres g ON mg.genre_id = g.id
			WHERE mg.manga_id = m.id
		), '[]'::json) as genres
	FROM manga m
	LEFT JOIN manga_stats s ON m.id = s.manga_id
	WHERE m.search_vector @@ websearch_to_tsquery('english', $1)
	`
	}

	// Add filters
	query, args, err := s.buildSearchQuery(baseQuery, req)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid search parameters: %v", err)
	}

	// Add ordering and limit (avoid fmt.Sprintf on SQL with % to prevent format string issues)
	limitParam := len(args) + 1
	if useShortMatch {
		query = query + fmt.Sprintf(" ORDER BY m.created_at DESC LIMIT $%d", limitParam)
	} else {
		query = query + fmt.Sprintf(" ORDER BY relevance_score DESC, m.created_at DESC LIMIT $%d", limitParam)
	}
	args = append(args, req.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		logger.Errorf("StreamSearch query failed: %v", err)
		return status.Errorf(codes.Internal, "search failed: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		mangaResp, err := s.scanMangaRow(rows)
		if err != nil {
			logger.Warnf("StreamSearch scan error: %v", err)
			continue
		}

		if err := stream.Send(mangaResp); err != nil {
			return status.Errorf(codes.Aborted, "stream aborted: %v", err)
		}
		count++
	}

	logger.Infof("StreamSearch completed: %d results streamed", count)
	return nil
}

// SearchManga performs standard paginated search
func (s *MangaServiceServer) SearchManga(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	logger := logrus.StandardLogger()

	// Validate request
	if err := s.validateSearchRequest(req); err != nil {
		return nil, err
	}

	// Handle empty query
	if strings.TrimSpace(req.Query) == "" {
		return s.getRecentManga(ctx, req.Limit, req.Offset)
	}

	searchTerm := strings.ToLower(strings.TrimSpace(req.Query))
	if len(searchTerm) < 2 {
		return s.getRecentManga(ctx, req.Limit, req.Offset)
	}
	useShortMatch := len(searchTerm) < 3

	var baseQuery string
	if useShortMatch {
		baseQuery = `
	SELECT 
		m.id, m.title, m.description, m.cover_url, m.status,
		0 as relevance_score,
		COALESCE(s.comment_count, 0) as comment_count,
		COALESCE(s.chat_count, 0) as chat_count,
		COALESCE(s.weekly_score, 0) as weekly_score,
		m.created_at,
		COALESCE((
			SELECT json_agg(json_build_object('id', g.id, 'name', g.name) ORDER BY g.name)
			FROM manga_genres mg
			INNER JOIN genres g ON mg.genre_id = g.id
			WHERE mg.manga_id = m.id
		), '[]'::json) as genres
	FROM manga m
	LEFT JOIN manga_stats s ON m.id = s.manga_id
	WHERE LOWER(m.title) LIKE '%' || $1 || '%'
	`
	} else {
		// Build base query
		baseQuery = `
	SELECT 
		m.id, m.title, m.description, m.cover_url, m.status,
		ts_rank_cd(m.search_vector, websearch_to_tsquery('english', $1)) as relevance_score,
		COALESCE(s.comment_count, 0) as comment_count,
		COALESCE(s.chat_count, 0) as chat_count,
		COALESCE(s.weekly_score, 0) as weekly_score,
		m.created_at,
		COALESCE((
			SELECT json_agg(json_build_object('id', g.id, 'name', g.name) ORDER BY g.name)
			FROM manga_genres mg
			INNER JOIN genres g ON mg.genre_id = g.id
			WHERE mg.manga_id = m.id
		), '[]'::json) as genres
	FROM manga m
	LEFT JOIN manga_stats s ON m.id = s.manga_id
	WHERE m.search_vector @@ websearch_to_tsquery('english', $1)
	`
	}

	// Build full query with filters
	query, args, err := s.buildSearchQuery(baseQuery, req)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid search parameters: %v", err)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM (" + query + ") as count"
	var total int32
	err = s.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		logger.Errorf("SearchManga count query failed: %v", err)
		return nil, status.Errorf(codes.Internal, "count query failed: %v", err)
	}

	// Add ordering, limit, offset (avoid fmt.Sprintf on SQL with % to prevent format string issues)
	limitParam := len(args) + 1
	offsetParam := len(args) + 2
	if useShortMatch {
		query = query + fmt.Sprintf(" ORDER BY m.created_at DESC LIMIT $%d OFFSET $%d", limitParam, offsetParam)
	} else {
		query = query + fmt.Sprintf(" ORDER BY relevance_score DESC, m.created_at DESC LIMIT $%d OFFSET $%d", limitParam, offsetParam)
	}
	args = append(args, req.Limit, req.Offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		logger.Errorf("SearchManga query failed: %v", err)
		return nil, status.Errorf(codes.Internal, "search query failed: %v", err)
	}
	defer rows.Close()

	var results []*pb.MangaResponse
	for rows.Next() {
		mangaResp, err := s.scanMangaRow(rows)
		if err != nil {
			logger.Warnf("SearchManga scan error: %v", err)
			continue
		}
		results = append(results, mangaResp)
	}

	hasMore := int32(req.Offset+req.Limit) < total
	logger.Infof("SearchManga completed: %d/%d results returned", len(results), total)

	return &pb.SearchResponse{
		Manga:   results,
		Total:   total,
		Limit:   req.Limit,
		Offset:  req.Offset,
		HasMore: hasMore,
	}, nil
}

// GetManga retrieves a single manga by ID
func (s *MangaServiceServer) GetManga(ctx context.Context, req *pb.GetMangaRequest) (*pb.MangaResponse, error) {
	if req.MangaId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "manga_id is required")
	}

	// Get manga with genres
	mangaWithGenres, err := s.mangaRepo.GetWithGenres(ctx, req.MangaId)
	if err != nil {
		if err.Error() == "manga not found" {
			return nil, status.Errorf(codes.NotFound, "manga not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get manga: %v", err)
	}

	// Get stats
	stats, err := s.statsRepo.GetByMangaID(ctx, req.MangaId)
	if err != nil {
		// Don't fail if stats are missing, use defaults
		stats = &models.MangaStats{
			MangaID: req.MangaId,
		}
	}

	// Convert genres
	var genres []*pb.Genre
	for _, genre := range mangaWithGenres.Genres {
		genres = append(genres, &pb.Genre{
			Id:   genre.ID,
			Name: genre.Name,
		})
	}

	// Create response with all fields
	return &pb.MangaResponse{
		Id:            mangaWithGenres.Manga.ID,
		Title:         mangaWithGenres.Manga.Title,
		Description:   mangaWithGenres.Manga.Description,
		CoverUrl:      mangaWithGenres.Manga.CoverURL,
		Status:        string(mangaWithGenres.Manga.Status),
		Genres:        genres,
		CommentCount:  int32(stats.CommentCount),
		ChatCount:     int32(stats.ChatCount),
		WeeklyScore:   int32(stats.WeeklyScore),
		CreatedAt:     timestamppb.New(mangaWithGenres.Manga.CreatedAt),
		RelevanceScore: 1.0, // Perfect match for single manga
	}, nil
}

// GetTrendingManga returns hot manga based on weekly_score
func (s *MangaServiceServer) GetTrendingManga(ctx context.Context, req *pb.TrendingRequest) (*pb.TrendingResponse, error) {
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 50 {
		req.Limit = 50
	}

	// Get top manga by weekly score from stats repository
	statsList, _, err := s.statsRepo.GetTopByWeeklyScore(ctx, int(req.Limit), 0)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get trending manga: %v", err)
	}

	// Convert to protobuf responses
	var results []*pb.MangaResponse
	for _, stats := range statsList {
		mangaWithGenres, err := s.mangaRepo.GetWithGenres(ctx, stats.MangaID)
		if err != nil {
			continue // Skip if genres fail to load
		}

		// Convert genres
		var genres []*pb.Genre
		for _, genre := range mangaWithGenres.Genres {
			genres = append(genres, &pb.Genre{
				Id:   genre.ID,
				Name: genre.Name,
			})
		}

		results = append(results, &pb.MangaResponse{
			Id:            stats.MangaID,
			Title:         mangaWithGenres.Manga.Title,
			CoverUrl:      mangaWithGenres.Manga.CoverURL,
			Status:        string(mangaWithGenres.Manga.Status),
			Genres:        genres,
			RelevanceScore: float32(stats.WeeklyScore) / 1000.0, // Normalize score
			CommentCount:  int32(stats.CommentCount),
			ChatCount:     int32(stats.ChatCount),
			WeeklyScore:   int32(stats.WeeklyScore),
			CreatedAt:     timestamppb.New(mangaWithGenres.Manga.CreatedAt),
		})
	}

	return &pb.TrendingResponse{
		Manga: results,
	}, nil
}

// AutoSuggest provides search suggestions as user types
func (s *MangaServiceServer) AutoSuggest(ctx context.Context, req *pb.AutoSuggestRequest) (*pb.AutoSuggestResponse, error) {
	if len(req.Prefix) < 2 {
		return &pb.AutoSuggestResponse{Suggestions: []string{}}, nil
	}

	if req.Limit <= 0 {
		req.Limit = 5
	}
	if req.Limit > 10 {
		req.Limit = 10
	}

	// Use PostgreSQL prefix matching with tsvector
	prefix := strings.ReplaceAll(strings.ToLower(req.Prefix), "'", "''")
	query := `
	SELECT DISTINCT title
	FROM manga
	WHERE search_vector @@ to_tsquery('english', $1 || ':*')
	ORDER BY created_at DESC
	LIMIT $2
	`

	rows, err := s.pool.Query(ctx, query, prefix, req.Limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "auto-suggest query failed: %v", err)
	}
	defer rows.Close()

	var suggestions []string
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			continue
		}
		suggestions = append(suggestions, title)
	}

	return &pb.AutoSuggestResponse{
		Suggestions: suggestions,
	}, nil
}

// HealthCheck implements gRPC health checking
func (s *MangaServiceServer) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	// Check database connectivity
	if err := s.pool.Ping(ctx); err != nil {
		return &pb.HealthCheckResponse{
			Status:  pb.HealthStatus_NOT_SERVING,
			Message: fmt.Sprintf("database unreachable: %v", err),
		}, nil
	}

	// Check FTS functionality
	var count int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM manga 
		WHERE search_vector IS NOT NULL 
		LIMIT 1
	`).Scan(&count)
	if err != nil {
		return &pb.HealthCheckResponse{
			Status:  pb.HealthStatus_NOT_SERVING,
			Message: fmt.Sprintf("FTS index issue: %v", err),
		}, nil
	}

	return &pb.HealthCheckResponse{
		Status:  pb.HealthStatus_SERVING,
		Message: "OK",
		Version: "1.0.0",
		Uptime:  time.Now().Unix(),
	}, nil
}

// Helper methods

func (s *MangaServiceServer) validateSearchRequest(req *pb.SearchRequest) error {
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
	return nil
}

func (s *MangaServiceServer) buildSearchQuery(baseQuery string, req *pb.SearchRequest) (string, []interface{}, error) {
	args := []interface{}{strings.ToLower(strings.TrimSpace(req.Query))}
	paramCount := 1
	filters := []string{}

	// Genre filters
	if len(req.GenreIds) > 0 {
		genrePlaceholders := make([]string, len(req.GenreIds))
		for i, genreID := range req.GenreIds {
			paramCount++
			genrePlaceholders[i] = fmt.Sprintf("$%d", paramCount)
			args = append(args, genreID)
		}
		filters = append(filters, fmt.Sprintf("m.id IN (SELECT manga_id FROM manga_genres WHERE genre_id IN (%s))", strings.Join(genrePlaceholders, ",")))
	}

	// Status filter
	if req.Status != "" {
		paramCount++
		args = append(args, req.Status)
		filters = append(filters, fmt.Sprintf("m.status = $%d", paramCount))
	}

	// Combine filters
	if len(filters) > 0 {
		baseQuery += " AND " + strings.Join(filters, " AND ")
	}

	return baseQuery, args, nil
}

func (s *MangaServiceServer) scanMangaRow(rows pgx.Rows) (*pb.MangaResponse, error) {
	var mangaID, title, status string
	var description, coverURL pgtype.Text
	var relevanceScore float64
	var commentCount, chatCount, weeklyScore int
	var createdAt time.Time
	var genresJSON []byte

	if err := rows.Scan(
		&mangaID,
		&title,
		&description,
		&coverURL,
		&status,
		&relevanceScore,
		&commentCount,
		&chatCount,
		&weeklyScore,
		&createdAt,
		&genresJSON,
	); err != nil {
		return nil, err
	}

	descriptionStr := ""
	if description.Valid {
		descriptionStr = description.String
	}
	coverURLStr := ""
	if coverURL.Valid {
		coverURLStr = coverURL.String
	}

	// Parse genres
	var genres []*pb.Genre
	if len(genresJSON) > 0 {
		var genreData []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(genresJSON, &genreData); err == nil {
			genres = make([]*pb.Genre, 0, len(genreData))
			for _, g := range genreData {
				genres = append(genres, &pb.Genre{Id: g.ID, Name: g.Name})
			}
		}
	}

	return &pb.MangaResponse{
		Id:            mangaID,
		Title:         title,
		Description:   descriptionStr,
		CoverUrl:      coverURLStr,
		Status:        status,
		Genres:        genres,
		RelevanceScore: float32(relevanceScore),
		CommentCount:  int32(commentCount),
		ChatCount:     int32(chatCount),
		WeeklyScore:   int32(weeklyScore),
		CreatedAt:     timestamppb.New(createdAt),
	}, nil
}

func (s *MangaServiceServer) getRecentManga(ctx context.Context, limit, offset int32) (*pb.SearchResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Count total
	var total int32
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM manga").Scan(&total); err != nil {
		return nil, status.Errorf(codes.Internal, "count query failed: %v", err)
	}

	query := `
	SELECT 
		m.id, m.title, m.description, m.cover_url, m.status,
		0 as relevance_score,
		COALESCE(s.comment_count, 0) as comment_count,
		COALESCE(s.chat_count, 0) as chat_count,
		COALESCE(s.weekly_score, 0) as weekly_score,
		m.created_at,
		COALESCE((
			SELECT json_agg(json_build_object('id', g.id, 'name', g.name) ORDER BY g.name)
			FROM manga_genres mg
			INNER JOIN genres g ON mg.genre_id = g.id
			WHERE mg.manga_id = m.id
		), '[]'::json) as genres
	FROM manga m
	LEFT JOIN manga_stats s ON m.id = s.manga_id
	ORDER BY m.created_at DESC
	LIMIT $1 OFFSET $2
	`

	rows, err := s.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "recent query failed: %v", err)
	}
	defer rows.Close()

	var results []*pb.MangaResponse
	for rows.Next() {
		mangaResp, err := s.scanMangaRow(rows)
		if err != nil {
			continue
		}
		results = append(results, mangaResp)
	}

	hasMore := int32(offset+limit) < total
	return &pb.SearchResponse{
		Manga:   results,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: hasMore,
	}, nil
}

func (s *MangaServiceServer) streamRecentManga(ctx context.Context, stream pb.MangaService_StreamSearchServer, limit int32) error {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	query := `
	SELECT 
		m.id, m.title, m.description, m.cover_url, m.status,
		0 as relevance_score,
		COALESCE(s.comment_count, 0) as comment_count,
		COALESCE(s.chat_count, 0) as chat_count,
		COALESCE(s.weekly_score, 0) as weekly_score,
		m.created_at,
		COALESCE((
			SELECT json_agg(json_build_object('id', g.id, 'name', g.name) ORDER BY g.name)
			FROM manga_genres mg
			INNER JOIN genres g ON mg.genre_id = g.id
			WHERE mg.manga_id = m.id
		), '[]'::json) as genres
	FROM manga m
	LEFT JOIN manga_stats s ON m.id = s.manga_id
	ORDER BY m.created_at DESC
	LIMIT $1
	`

	rows, err := s.pool.Query(ctx, query, limit)
	if err != nil {
		return status.Errorf(codes.Internal, "recent query failed: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		mangaResp, err := s.scanMangaRow(rows)
		if err != nil {
			continue
		}
		if err := stream.Send(mangaResp); err != nil {
			return status.Errorf(codes.Aborted, "stream aborted: %v", err)
		}
	}

	return nil
}