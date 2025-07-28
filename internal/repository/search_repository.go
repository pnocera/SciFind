package repository

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"scifind-backend/internal/errors"
	"scifind-backend/internal/models"

	"gorm.io/gorm"
)

// searchRepository implements SearchRepository interface
type searchRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewSearchRepository creates a new search repository
func NewSearchRepository(db *gorm.DB, logger *slog.Logger) SearchRepository {
	return &searchRepository{
		db:     db,
		logger: logger,
	}
}

// CreateSearchHistory creates a search history entry
func (r *searchRepository) CreateSearchHistory(ctx context.Context, history *models.SearchHistory) error {
	if err := r.db.WithContext(ctx).Create(history).Error; err != nil {
		return errors.NewDatabaseError("create_search_history", err)
	}
	return nil
}

// GetSearchHistory retrieves search history for a user or globally
func (r *searchRepository) GetSearchHistory(ctx context.Context, userID *string, limit, offset int) ([]models.SearchHistory, error) {
	query := r.db.WithContext(ctx).Model(&models.SearchHistory{})
	
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	
	var history []models.SearchHistory
	err := query.Order("requested_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&history).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_search_history", err)
	}
	
	return history, nil
}

// GetPopularQueries returns popular search queries since a given time
func (r *searchRepository) GetPopularQueries(ctx context.Context, since time.Time, limit int) ([]QueryStats, error) {
	var queries []struct {
		Query       string
		Count       int64
		LastQueried time.Time
	}
	
	err := r.db.WithContext(ctx).
		Model(&models.SearchHistory{}).
		Select("query, COUNT(*) as count, MAX(requested_at) as last_queried").
		Where("requested_at >= ?", since).
		Group("query").
		Order("count DESC").
		Limit(limit).
		Scan(&queries).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_popular_queries", err)
	}
	
	stats := make([]QueryStats, len(queries))
	for i, q := range queries {
		stats[i] = QueryStats{
			Query:       q.Query,
			Count:       q.Count,
			LastQueried: q.LastQueried,
		}
	}
	
	return stats, nil
}

// GetUserSearchStats returns search statistics for a specific user
func (r *searchRepository) GetUserSearchStats(ctx context.Context, userID string) (*UserSearchStats, error) {
	var stats UserSearchStats
	stats.UserID = userID
	
	// Total queries
	err := r.db.WithContext(ctx).
		Model(&models.SearchHistory{}).
		Where("user_id = ?", userID).
		Count(&stats.TotalQueries).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_user_stats_total", err)
	}
	
	// Unique queries
	err = r.db.WithContext(ctx).
		Model(&models.SearchHistory{}).
		Select("COUNT(DISTINCT query)").
		Where("user_id = ?", userID).
		Scan(&stats.UniqueQueries).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_user_stats_unique", err)
	}
	
	// Last search
	var lastSearch models.SearchHistory
	err = r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("requested_at DESC").
		First(&lastSearch).Error
	
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.NewDatabaseError("get_user_stats_last", err)
	}
	
	if err != gorm.ErrRecordNotFound {
		stats.LastSearch = lastSearch.RequestedAt
	}
	
	// Top queries for this user
	var topQueries []struct {
		Query       string
		Count       int64
		LastQueried time.Time
	}
	
	err = r.db.WithContext(ctx).
		Model(&models.SearchHistory{}).
		Select("query, COUNT(*) as count, MAX(requested_at) as last_queried").
		Where("user_id = ?", userID).
		Group("query").
		Order("count DESC").
		Limit(10).
		Scan(&topQueries).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_user_stats_top_queries", err)
	}
	
	stats.TopQueries = make([]QueryStats, len(topQueries))
	for i, q := range topQueries {
		stats.TopQueries[i] = QueryStats{
			Query:       q.Query,
			Count:       q.Count,
			LastQueried: q.LastQueried,
		}
	}
	
	return &stats, nil
}

// GetCachedSearch retrieves cached search results
func (r *searchRepository) GetCachedSearch(ctx context.Context, queryHash string) (*models.SearchCache, error) {
	var cache models.SearchCache
	err := r.db.WithContext(ctx).
		First(&cache, "query_hash = ? AND expires_at > ?", queryHash, time.Now()).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Cache miss, not an error
		}
		return nil, errors.NewDatabaseError("get_cached_search", err)
	}
	
	// Update access information
	cache.IncrementAccess()
	r.db.WithContext(ctx).Save(&cache)
	
	return &cache, nil
}

// SetSearchCache stores search results in cache
func (r *searchRepository) SetSearchCache(ctx context.Context, cache *models.SearchCache) error {
	// Generate hash if not provided
	if cache.QueryHash == "" {
		cache.QueryHash = r.generateQueryHash(cache.Query, cache.Provider)
	}
	
	// Try to update existing cache entry first
	result := r.db.WithContext(ctx).
		Where("query_hash = ?", cache.QueryHash).
		Updates(cache)
	
	if result.Error != nil {
		return errors.NewDatabaseError("update_search_cache", result.Error)
	}
	
	// If no rows were affected, create new entry
	if result.RowsAffected == 0 {
		if err := r.db.WithContext(ctx).Create(cache).Error; err != nil {
			return errors.NewDatabaseError("create_search_cache", err)
		}
	}
	
	return nil
}

// InvalidateCache removes cache entries matching a pattern
func (r *searchRepository) InvalidateCache(ctx context.Context, pattern string) error {
	err := r.db.WithContext(ctx).
		Where("query ILIKE ?", pattern).
		Delete(&models.SearchCache{}).Error
	
	if err != nil {
		return errors.NewDatabaseError("invalidate_cache", err)
	}
	
	return nil
}

// CleanupExpiredCache removes expired cache entries
func (r *searchRepository) CleanupExpiredCache(ctx context.Context) error {
	err := r.db.WithContext(ctx).
		Where("expires_at <= ?", time.Now()).
		Delete(&models.SearchCache{}).Error
	
	if err != nil {
		return errors.NewDatabaseError("cleanup_expired_cache", err)
	}
	
	return nil
}

// GetCacheStats returns cache statistics
func (r *searchRepository) GetCacheStats(ctx context.Context) (*CacheStats, error) {
	var stats CacheStats
	
	// Total entries
	err := r.db.WithContext(ctx).
		Model(&models.SearchCache{}).
		Count(&stats.TotalEntries).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_cache_stats_total", err)
	}
	
	// Expired entries
	err = r.db.WithContext(ctx).
		Model(&models.SearchCache{}).
		Where("expires_at <= ?", time.Now()).
		Count(&stats.ExpiredEntries).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_cache_stats_expired", err)
	}
	
	// Calculate hit rate (this would need historical data)
	// For now, using access_count as a proxy
	var totalAccess, totalEntries int64
	var result struct {
		TotalAccess  int64 `gorm:"column:total_access"`
		TotalEntries int64 `gorm:"column:total_entries"`
	}

	err = r.db.WithContext(ctx).
		Model(&models.SearchCache{}).
		Select("SUM(access_count) as total_access, COUNT(*) as total_entries").
		Scan(&result).Error

	if err == nil {
		totalAccess = result.TotalAccess
		totalEntries = result.TotalEntries
	}
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_cache_stats_hit_rate", err)
	}
	
	if totalEntries > 0 {
		stats.HitRate = float64(totalAccess) / float64(totalEntries)
	}
	
	// Average age in hours
	var avgAge float64
	err = r.db.WithContext(ctx).
		Model(&models.SearchCache{}).
		Select("AVG(EXTRACT(EPOCH FROM (NOW() - created_at))/3600)").
		Scan(&avgAge).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_cache_stats_avg_age", err)
	}
	stats.AvgAge = avgAge
	
	// Size estimation (this would need actual size calculation)
	stats.SizeBytes = stats.TotalEntries * 1024 // Rough estimate
	
	return &stats, nil
}

// GetSearchSuggestions returns search suggestions based on query
func (r *searchRepository) GetSearchSuggestions(ctx context.Context, query string, limit int) ([]models.SearchSuggestion, error) {
	var suggestions []models.SearchSuggestion
	
	// Get suggestions from search history
	var historyQueries []struct {
		Query string
		Count int64
	}
	
	err := r.db.WithContext(ctx).
		Model(&models.SearchHistory{}).
		Select("query, COUNT(*) as count").
		Where("query ILIKE ?", query+"%").
		Group("query").
		Order("count DESC").
		Limit(limit).
		Scan(&historyQueries).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_search_suggestions", err)
	}
	
	for _, hq := range historyQueries {
		suggestions = append(suggestions, models.SearchSuggestion{
			Text:  hq.Query,
			Score: float64(hq.Count),
			Type:  "query",
		})
	}
	
	// Could also add suggestions from paper titles, author names, etc.
	
	return suggestions, nil
}

// UpdateSearchSuggestions updates search suggestions based on query performance
func (r *searchRepository) UpdateSearchSuggestions(ctx context.Context, query string, resultCount int) error {
	// This could store suggestion weights/scores in a separate table
	// For now, we'll just record it in search history
	
	history := &models.SearchHistory{
		ID:          "suggestion_" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Query:       query,
		ResultCount: resultCount,
		Duration:    0, // Not tracking duration for suggestions
		Providers:   []string{},
		RequestedAt: time.Now(),
	}
	
	return r.CreateSearchHistory(ctx, history)
}

// GetSearchAnalytics returns search analytics for a time period
func (r *searchRepository) GetSearchAnalytics(ctx context.Context, from, to time.Time) (*SearchAnalytics, error) {
	var analytics SearchAnalytics
	
	// Total searches
	err := r.db.WithContext(ctx).
		Model(&models.SearchHistory{}).
		Where("requested_at BETWEEN ? AND ?", from, to).
		Count(&analytics.TotalSearches).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_analytics_total", err)
	}
	
	// Unique queries
	err = r.db.WithContext(ctx).
		Model(&models.SearchHistory{}).
		Select("COUNT(DISTINCT query)").
		Where("requested_at BETWEEN ? AND ?", from, to).
		Scan(&analytics.UniqueQueries).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_analytics_unique", err)
	}
	
	// Average response time
	var avgDuration float64
	err = r.db.WithContext(ctx).
		Model(&models.SearchHistory{}).
		Select("AVG(duration)").
		Where("requested_at BETWEEN ? AND ? AND duration > 0", from, to).
		Scan(&avgDuration).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_analytics_avg_duration", err)
	}
	analytics.AvgResponseTime = avgDuration
	
	// Top queries
	topQueries, err := r.GetPopularQueries(ctx, from, 10)
	if err != nil {
		return nil, err
	}
	analytics.TopQueries = topQueries
	
	// Provider usage stats would need more complex queries
	// For now, returning empty slice
	analytics.ProviderUsage = []ProviderUsageStats{}
	
	// Cache hit rate and error rate would need additional tracking
	analytics.CacheHitRate = 0.0
	analytics.ErrorRate = 0.0
	
	return &analytics, nil
}

// GetProviderPerformance returns performance metrics for a specific provider
func (r *searchRepository) GetProviderPerformance(ctx context.Context, provider string, from, to time.Time) (*ProviderPerformance, error) {
	var perf ProviderPerformance
	perf.Provider = provider
	
	// This would require more detailed tracking of provider-specific metrics
	// For now, returning basic structure with zero values
	perf.TotalRequests = 0
	perf.SuccessRate = 0.0
	perf.AvgResponseTime = 0.0
	perf.TotalResults = 0
	perf.AvgResultsPerRequest = 0.0
	
	return &perf, nil
}

// Helper methods

// generateQueryHash generates a hash for a query and provider combination
func (r *searchRepository) generateQueryHash(query, provider string) string {
	data := fmt.Sprintf("%s:%s", query, provider)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// generateSearchCacheID generates a unique ID for search cache entries
func (r *searchRepository) generateSearchCacheID() string {
	return "cache_" + fmt.Sprintf("%d", time.Now().UnixNano())
}

// serializeResults serializes search results to JSON string
func (r *searchRepository) serializeResults(results interface{}) (string, error) {
	data, err := json.Marshal(results)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// deserializeResults deserializes JSON string to search results
func (r *searchRepository) deserializeResults(data string, target interface{}) error {
	return json.Unmarshal([]byte(data), target)
}