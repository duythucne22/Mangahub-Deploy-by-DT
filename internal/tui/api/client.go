package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"mangahub/pkg/models"
)

// Client handles HTTP API communication
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	userID     string // Store current user ID
}

// NewClient creates a new API client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetToken sets the authentication token
func (c *Client) SetToken(token string) {
	c.token = token
}

// GetToken returns the current authentication token
func (c *Client) GetToken() string {
	return c.token
}

// SetUserID sets the current user ID
func (c *Client) SetUserID(userID string) {
	c.userID = userID
}

// GetUserID returns the current user ID
func (c *Client) GetUserID() string {
	return c.userID
}

// doRequest performs an HTTP request with common handling
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// decodeResponse decodes JSON response into target
func decodeResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

type apiResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// decodeAPIResponse decodes the APIResponse envelope and unmarshals the data field into target
func decodeAPIResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success {
		if apiResp.Error != "" {
			return fmt.Errorf("%s", apiResp.Error)
		}
		return fmt.Errorf("request failed")
	}

	if target != nil && len(apiResp.Data) > 0 {
		if err := json.Unmarshal(apiResp.Data, target); err != nil {
			return fmt.Errorf("failed to decode response data: %w", err)
		}
	}

	return nil
}

// Auth endpoints

// Register creates a new user account
func (c *Client) Register(ctx context.Context, username, email, password string) (*models.LoginResponse, error) {
	body := map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	}

	resp, err := c.doRequest(ctx, "POST", "/auth/register", body)
	if err != nil {
		return nil, err
	}

	var registerData struct {
		User models.User `json:"user"`
	}
	if err := decodeAPIResponse(resp, &registerData); err != nil {
		return nil, err
	}

	loginResp, err := c.Login(ctx, username, password)
	if err != nil {
		return nil, err
	}

	return loginResp, nil
}

// Login authenticates a user
func (c *Client) Login(ctx context.Context, username, password string) (*models.LoginResponse, error) {
	body := map[string]string{
		"username": username,
		"password": password,
	}

	resp, err := c.doRequest(ctx, "POST", "/auth/login", body)
	if err != nil {
		return nil, err
	}

	var loginResp models.LoginResponse
	if err := decodeAPIResponse(resp, &loginResp); err != nil {
		return nil, err
	}

	c.token = loginResp.Token
	c.userID = loginResp.User.ID
	return &loginResp, nil
}


// Manga endpoints

// ListManga retrieves a paginated list of manga with optional genre filter
func (c *Client) ListManga(ctx context.Context, page, limit int) (*models.PaginatedResponse[models.Manga], error) {
	path := fmt.Sprintf("/manga?page=%d&limit=%d", page, limit)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var apiResult models.MangaListResponse
	if err := decodeAPIResponse(resp, &apiResult); err != nil {
		return nil, err
	}

	manga := make([]models.Manga, len(apiResult.Data))
	for i, item := range apiResult.Data {
		manga[i] = item.Manga
	}

	result := models.PaginatedResponse[models.Manga]{
		Data: manga,
		Meta: models.PaginationMeta{
			Total:   apiResult.Total,
			Limit:   apiResult.Limit,
			Offset:  apiResult.Offset,
			HasMore: apiResult.HasMore,
		},
	}

	return &result, nil
}

// ListMangaByGenre retrieves manga filtered by genre
func (c *Client) ListMangaByGenre(ctx context.Context, genre string, page, limit int) (*models.PaginatedResponse[models.Manga], error) {
	path := fmt.Sprintf("/manga?genre=%s&page=%d&limit=%d", genre, page, limit)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	
	if err != nil {
		return nil, err
	}

	var apiResult models.MangaListResponse
	if err := decodeAPIResponse(resp, &apiResult); err != nil {
		return nil, err
	}

	manga := make([]models.Manga, len(apiResult.Data))
	for i, item := range apiResult.Data {
		manga[i] = item.Manga
	}

	result := models.PaginatedResponse[models.Manga]{
		Data: manga,
		Meta: models.PaginationMeta{
			Total:   apiResult.Total,
			Limit:   apiResult.Limit,
			Offset:  apiResult.Offset,
			HasMore: apiResult.HasMore,
		},
	}

	return &result, nil
}

// GetManga retrieves a specific manga by ID
func (c *Client) GetManga(ctx context.Context, id string) (*models.Manga, error) {
	path := fmt.Sprintf("/manga/%s", id)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var manga models.MangaWithGenres
	if err := decodeAPIResponse(resp, &manga); err != nil {
		return nil, err
	}

	return &manga.Manga, nil
}

// SearchManga searches for manga by query
func (c *Client) SearchManga(ctx context.Context, query string, page, limit int) (*models.PaginatedResponse[models.Manga], error) {
	path := fmt.Sprintf("/manga/search?q=%s&page=%d&limit=%d", query, page, limit)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var apiResult models.MangaListResponse
	if err := decodeAPIResponse(resp, &apiResult); err != nil {
		return nil, err
	}

	manga := make([]models.Manga, len(apiResult.Data))
	for i, item := range apiResult.Data {
		manga[i] = item.Manga
	}

	result := models.PaginatedResponse[models.Manga]{
		Data: manga,
		Meta: models.PaginationMeta{
			Total:   apiResult.Total,
			Limit:   apiResult.Limit,
			Offset:  apiResult.Offset,
			HasMore: apiResult.HasMore,
		},
	}

	return &result, nil
}

// GetTrending retrieves trending manga
func (c *Client) GetTrending(ctx context.Context, limit int) ([]models.Manga, error) {
	path := fmt.Sprintf("/manga/trending?limit=%d", limit)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	// API returns RankedMangaResponse with pagination metadata
	var result models.RankedMangaResponse
	if err := decodeAPIResponse(resp, &result); err != nil {
		return nil, err
	}

	// Extract manga from RankedManga entries
	manga := make([]models.Manga, len(result.Data))
	for i, ranked := range result.Data {
		manga[i] = ranked.Manga
	}

	return manga, nil
}

// Comment endpoints

// ListComments retrieves comments for a manga
func (c *Client) ListComments(ctx context.Context, mangaID string, page, limit int) (*models.PaginatedResponse[models.Comment], error) {
	path := fmt.Sprintf("/manga/%s/comments?page=%d&limit=%d", mangaID, page, limit)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var apiResult models.CommentListResponse
	if err := decodeAPIResponse(resp, &apiResult); err != nil {
		return nil, err
	}

	comments := make([]models.Comment, len(apiResult.Data))
	for i, item := range apiResult.Data {
		comments[i] = models.Comment{
			ID:        item.ID,
			MangaID:   item.MangaID,
			UserID:    item.User.ID,
			Content:   item.Content,
			LikeCount: item.LikeCount,
			CreatedAt: item.CreatedAt,
		}
	}

	result := models.PaginatedResponse[models.Comment]{
		Data: comments,
		Meta: models.PaginationMeta{
			Total:   apiResult.Total,
			Limit:   apiResult.Limit,
			Offset:  apiResult.Offset,
			HasMore: apiResult.HasMore,
		},
	}

	return &result, nil
}

// CreateComment creates a new comment on a manga
func (c *Client) CreateComment(ctx context.Context, mangaID, content string, rating *int) (*models.Comment, error) {
	body := map[string]interface{}{
		"manga_id": mangaID,
		"content":  content,
	}
	if rating != nil {
		body["rating"] = *rating
	}

	path := fmt.Sprintf("/manga/%s/comments", mangaID)
	resp, err := c.doRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}

	var commentResp models.CommentResponse
	if err := decodeAPIResponse(resp, &commentResp); err != nil {
		return nil, err
	}

	comment := models.Comment{
		ID:        commentResp.ID,
		MangaID:   commentResp.MangaID,
		UserID:    commentResp.User.ID,
		Content:   commentResp.Content,
		LikeCount: commentResp.LikeCount,
		CreatedAt: commentResp.CreatedAt,
	}

	return &comment, nil
}

// DeleteComment deletes a comment
func (c *Client) DeleteComment(ctx context.Context, mangaID, commentID string) error {
	path := fmt.Sprintf("/manga/%s/comments/%s", mangaID, commentID)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete comment (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Activity endpoints

// GetRecentActivity retrieves recent user activity
func (c *Client) GetRecentActivity(ctx context.Context, limit int) ([]models.ActivityResponse, error) {
	path := fmt.Sprintf("/activity/global?limit=%d", limit)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	// API returns ActivityFeedResponse with pagination metadata
	var result models.ActivityFeedResponse
	if err := decodeResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetUserActivity retrieves activity for a specific user
func (c *Client) GetUserActivity(ctx context.Context, userID string, page, limit int) (*models.ActivityFeedResponse, error) {
	path := fmt.Sprintf("/activity/user/%s?page=%d&limit=%d", userID, page, limit)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result models.ActivityFeedResponse
	if err := decodeResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Statistics endpoints

// GetUserStatistics retrieves statistics for a user
func (c *Client) GetUserStatistics(ctx context.Context, userID string) (*models.UserStatistics, error) {
	path := fmt.Sprintf("/statistics/user/%s", userID)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var stats models.UserStatistics
	if err := decodeAPIResponse(resp, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetTopManga retrieves top manga leaderboard by weekly score
func (c *Client) GetTopManga(ctx context.Context, limit int) (*models.RankedMangaResponse, error) {
	path := fmt.Sprintf("/stats/top?limit=%d", limit)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result models.RankedMangaResponse
	if err := decodeAPIResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}