package services

import (
	"context"
	"time"

	"scifind-backend/internal/models"
)


// SearchServiceInterface defines the contract for search service
type SearchServiceInterface interface {
	Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error)
	GetPaper(ctx context.Context, providerName, paperID string) (*models.Paper, error)
	GetProviderStatus(ctx context.Context) (map[string]interface{}, error)
	GetProviderMetrics(ctx context.Context) (map[string]interface{}, error)
	ConfigureProvider(ctx context.Context, name string, config interface{}) error
	Health(ctx context.Context) error
}

// PaperServiceInterface defines the contract for paper service
type PaperServiceInterface interface {
	GetByID(ctx context.Context, id string) (*models.Paper, error)
	Create(ctx context.Context, paper *models.Paper) error
	Update(ctx context.Context, paper *models.Paper) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*models.Paper, int, error)
	Search(ctx context.Context, query string, limit, offset int) ([]*models.Paper, int, error)
	GetByProvider(ctx context.Context, provider, sourceID string) (*models.Paper, error)
	Health(ctx context.Context) error
}

// AnalyticsServiceInterface defines the contract for analytics service
type AnalyticsServiceInterface interface {
	GetSearchMetrics(ctx context.Context, from, to time.Time) (*SearchMetrics, error)
	GetPopularQueries(ctx context.Context, limit int, from, to time.Time) ([]*PopularQuery, error)
	GetProviderPerformance(ctx context.Context, from, to time.Time) (map[string]*ProviderMetrics, error)
	GetUserActivity(ctx context.Context, userID string, from, to time.Time) (*UserActivity, error)
	RecordEvent(ctx context.Context, event *AnalyticsEvent) error
	Health(ctx context.Context) error
}

// HealthServiceInterface defines the contract for health service
type HealthServiceInterface interface {
	Health(ctx context.Context) error
	DatabaseHealth(ctx context.Context) error
	MessagingHealth(ctx context.Context) error
	ExternalServicesHealth(ctx context.Context) map[string]error
	GetSystemInfo(ctx context.Context) (*SystemInfo, error)
}

// AuthorServiceInterface defines the contract for author service
type AuthorServiceInterface interface {
	GetByID(ctx context.Context, id string) (*models.Author, error)
	Create(ctx context.Context, author *models.Author) error
	Update(ctx context.Context, author *models.Author) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*models.Author, int, error)
	Search(ctx context.Context, query string, limit, offset int) ([]*models.Author, int, error)
	GetPapers(ctx context.Context, authorID string, limit, offset int) ([]*models.Paper, int, error)
	Health(ctx context.Context) error
}

// Analytics data structures
type SearchMetrics struct {
	TotalSearches     int                `json:"total_searches"`
	UniqueUsers       int                `json:"unique_users"`
	AverageResultTime time.Duration      `json:"average_result_time"`
	SuccessRate       float64            `json:"success_rate"`
	PopularProviders  map[string]int     `json:"popular_providers"`
	SearchesByHour    []HourlyMetric     `json:"searches_by_hour"`
}


type ProviderMetrics struct {
	Name            string        `json:"name"`
	TotalRequests   int           `json:"total_requests"`
	SuccessRate     float64       `json:"success_rate"`
	AverageLatency  time.Duration `json:"average_latency"`
	ErrorRate       float64       `json:"error_rate"`
	LastHealthCheck time.Time     `json:"last_health_check"`
	IsHealthy       bool          `json:"is_healthy"`
}

type UserActivity struct {
	UserID        string             `json:"user_id"`
	SearchCount   int                `json:"search_count"`
	UniqueQueries int                `json:"unique_queries"`
	FavoriteTopics []string          `json:"favorite_topics"`
	ActivityByDay  []DailyActivity   `json:"activity_by_day"`
	LastActive     time.Time         `json:"last_active"`
}

type HourlyMetric struct {
	Hour  int `json:"hour"`
	Count int `json:"count"`
}

type DailyActivity struct {
	Date        time.Time `json:"date"`
	SearchCount int       `json:"search_count"`
}

type AnalyticsEvent struct {
	Type      string                 `json:"type"`
	UserID    string                 `json:"user_id,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

type SystemInfo struct {
	Version    string            `json:"version"`
	Uptime     time.Duration     `json:"uptime"`
	Memory     MemoryInfo        `json:"memory"`
	Database   DatabaseInfo      `json:"database"`
	Services   map[string]bool   `json:"services"`
	Timestamp  time.Time         `json:"timestamp"`
}

type MemoryInfo struct {
	Allocated uint64 `json:"allocated"`
	Total     uint64 `json:"total"`
	System    uint64 `json:"system"`
	GCRuns    uint32 `json:"gc_runs"`
}

type DatabaseInfo struct {
	Connected     bool              `json:"connected"`
	Type          string            `json:"type"`
	Version       string            `json:"version,omitempty"`
	Connections   map[string]int    `json:"connections"`
	Stats         map[string]interface{} `json:"stats,omitempty"`
}