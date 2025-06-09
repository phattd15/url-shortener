package handlers

import (
	"net/http"
	"net/url"
	"time"

	"url-shortener/cache"
	"url-shortener/database"
	"url-shortener/models"
	"url-shortener/utils"

	"github.com/gin-gonic/gin"
)

// ShortenURL godoc
// @Summary Create a short URL
// @Description Create a short URL from a long URL with optional expiration
// @Tags URL Shortener
// @Accept json
// @Produce json
// @Param request body models.ShortenRequest true "URL to shorten"
// @Success 201 {object} models.ShortenResponse
// @Success 200 {object} models.ShortenResponse "URL already exists"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /shorten [post]
func ShortenURL(c *gin.Context) {
	var request models.ShortenRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate URL
	if !isValidURL(request.URL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL format"})
		return
	}

	// Check cache first for existing URL
	if shortCode, err := cache.GetShortCodeForOriginalURL(request.URL); err == nil {
		// Found in cache, get the full URL data
		if urlData, err := cache.GetURLMapping(shortCode); err == nil {
			shortURL := buildShortURL(c, urlData.ShortCode)
			response := models.ShortenResponse{
				ShortURL:    shortURL,
				OriginalURL: urlData.OriginalURL,
				ShortCode:   urlData.ShortCode,
				ExpiresAt:   urlData.ExpiresAt,
			}
			c.JSON(http.StatusOK, response)
			return
		}
	}

	// Check database if not in cache
	var existingURL models.URL
	if err := database.DB.Where("original_url = ?", request.URL).First(&existingURL).Error; err == nil {
		// URL already exists in database, cache it and return
		cache.CacheURLMapping(existingURL.ShortCode, &existingURL)
		cache.CacheOriginalURLMapping(existingURL.OriginalURL, existingURL.ShortCode)

		shortURL := buildShortURL(c, existingURL.ShortCode)
		response := models.ShortenResponse{
			ShortURL:    shortURL,
			OriginalURL: existingURL.OriginalURL,
			ShortCode:   existingURL.ShortCode,
			ExpiresAt:   existingURL.ExpiresAt,
		}
		c.JSON(http.StatusOK, response)
		return
	}

	// Generate short code
	shortCode := utils.GenerateShortCode()

	// Create URL record
	urlRecord := models.URL{
		OriginalURL: request.URL,
		ShortCode:   shortCode,
		ClickCount:  0,
	}

	// Set expiration if provided
	if request.ExpiresIn > 0 {
		expiresAt := time.Now().AddDate(0, 0, request.ExpiresIn)
		urlRecord.ExpiresAt = &expiresAt
	}

	// Save to database
	if err := database.DB.Create(&urlRecord).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create short URL"})
		return
	}

	// Cache the new URL mapping
	cache.CacheURLMapping(urlRecord.ShortCode, &urlRecord)
	cache.CacheOriginalURLMapping(urlRecord.OriginalURL, urlRecord.ShortCode)

	// Build response
	shortURL := buildShortURL(c, shortCode)
	response := models.ShortenResponse{
		ShortURL:    shortURL,
		OriginalURL: urlRecord.OriginalURL,
		ShortCode:   urlRecord.ShortCode,
		ExpiresAt:   urlRecord.ExpiresAt,
	}

	c.JSON(http.StatusCreated, response)
}

// RedirectURL godoc
// @Summary Redirect to original URL
// @Description Redirect to the original URL using the short code and increment click count
// @Tags URL Shortener
// @Param shortCode path string true "Short code"
// @Success 301 "Redirects to original URL"
// @Failure 404 {object} map[string]string "Short URL not found"
// @Failure 410 {object} map[string]string "Short URL has expired"
// @Router /{shortCode} [get]
func RedirectURL(c *gin.Context) {
	shortCode := c.Param("shortCode")

	// Try cache first
	var urlRecord *models.URL
	var err error

	if cachedURL, cacheErr := cache.GetURLMapping(shortCode); cacheErr == nil {
		urlRecord = cachedURL
	} else {
		// Cache miss, check database
		var dbURL models.URL
		if err = database.DB.Where("short_code = ?", shortCode).First(&dbURL).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
			return
		}
		urlRecord = &dbURL
		// Cache the result for next time
		cache.CacheURLMapping(shortCode, urlRecord)
	}

	// Check if URL has expired
	if urlRecord.ExpiresAt != nil && urlRecord.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusGone, gin.H{"error": "Short URL has expired"})
		return
	}

	// Increment click count in cache (async)
	go func() {
		cache.IncrementClickCount(shortCode)
		// Also update in database (less frequently - could be batched)
		database.DB.Model(urlRecord).Update("click_count", urlRecord.ClickCount+1)
		// Invalidate stats cache since click count changed
		cache.InvalidateCache(shortCode)
	}()

	// Redirect to original URL
	c.Redirect(http.StatusMovedPermanently, urlRecord.OriginalURL)
}

// GetURLStats godoc
// @Summary Get URL statistics
// @Description Get statistics for a shortened URL including click count and creation date
// @Tags URL Shortener
// @Produce json
// @Param shortCode path string true "Short code"
// @Success 200 {object} models.StatsResponse
// @Failure 404 {object} map[string]string "Short URL not found"
// @Router /stats/{shortCode} [get]
func GetURLStats(c *gin.Context) {
	shortCode := c.Param("shortCode")

	// Try cache first
	if cachedStats, err := cache.GetURLStats(shortCode); err == nil {
		c.JSON(http.StatusOK, cachedStats)
		return
	}

	// Cache miss, get from database
	var urlRecord models.URL
	if err := database.DB.Where("short_code = ?", shortCode).First(&urlRecord).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
		return
	}

	// Get current click count from cache if available, otherwise use DB value
	clickCount := urlRecord.ClickCount
	if cachedClicks, err := cache.GetClickCount(shortCode); err == nil {
		clickCount = int(cachedClicks)
	}

	response := models.StatsResponse{
		OriginalURL: urlRecord.OriginalURL,
		ShortCode:   urlRecord.ShortCode,
		ClickCount:  clickCount,
		CreatedAt:   urlRecord.CreatedAt,
		ExpiresAt:   urlRecord.ExpiresAt,
	}

	// Cache the stats for a short time
	cache.CacheURLStats(shortCode, &response)

	c.JSON(http.StatusOK, response)
}

// HealthCheck godoc
// @Summary Health check
// @Description Check if the service is healthy and running
// @Tags System
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
func HealthCheck(c *gin.Context) {
	// Check database health
	sqlDB, err := database.DB.DB()
	dbHealthy := err == nil
	if dbHealthy {
		err = sqlDB.Ping()
		dbHealthy = err == nil
	}

	// Check Redis health
	redisHealthy := cache.IsRedisHealthy()

	response := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "url-shortener",
		"database":  map[string]bool{"healthy": dbHealthy},
		"cache":     map[string]bool{"healthy": redisHealthy},
	}

	// Return 503 if any critical service is down
	if !dbHealthy {
		response["status"] = "unhealthy"
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	// Redis is optional, so we don't fail if it's down
	if !redisHealthy {
		response["status"] = "degraded"
	}

	c.JSON(http.StatusOK, response)
}

func isValidURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func buildShortURL(c *gin.Context, shortCode string) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host + "/" + shortCode
}
