package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"url-shortener/models"

	"github.com/redis/go-redis/v9"
)

var (
	RedisClient *redis.Client
	ctx         = context.Background()
)

// Initialize Redis connection
func InitRedis() {
	addr := getEnv("REDIS_ADDR", "localhost:6379")
	password := getEnv("REDIS_PASSWORD", "")
	dbStr := getEnv("REDIS_DB", "0")

	db, err := strconv.Atoi(dbStr)
	if err != nil {
		log.Printf("Invalid REDIS_DB value, using default: %v", err)
		db = 0
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	_, err = RedisClient.Ping(ctx).Result()
	if err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
		log.Println("Continuing without cache...")
		RedisClient = nil
		return
	}

	log.Println("Redis connected successfully")
}

// Cache keys
const (
	URLMappingKey   = "url:mapping:%s"  // url:mapping:shortCode
	URLStatsKey     = "url:stats:%s"    // url:stats:shortCode
	OriginalURLKey  = "url:original:%s" // url:original:hashedURL
	DefaultCacheTTL = 24 * time.Hour    // 24 hours
	StatsCacheTTL   = 5 * time.Minute   // 5 minutes for stats
)

// Cache URL mapping (shortCode -> URL data)
func CacheURLMapping(shortCode string, urlData *models.URL) error {
	if RedisClient == nil {
		return nil // No-op if Redis is not available
	}

	key := fmt.Sprintf(URLMappingKey, shortCode)
	data, err := json.Marshal(urlData)
	if err != nil {
		return err
	}

	return RedisClient.Set(ctx, key, data, DefaultCacheTTL).Err()
}

// Get URL mapping from cache
func GetURLMapping(shortCode string) (*models.URL, error) {
	if RedisClient == nil {
		return nil, redis.Nil // Simulate cache miss if Redis not available
	}

	key := fmt.Sprintf(URLMappingKey, shortCode)
	data, err := RedisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var urlData models.URL
	err = json.Unmarshal([]byte(data), &urlData)
	if err != nil {
		return nil, err
	}

	return &urlData, nil
}

// Cache URL stats
func CacheURLStats(shortCode string, stats *models.StatsResponse) error {
	if RedisClient == nil {
		return nil
	}

	key := fmt.Sprintf(URLStatsKey, shortCode)
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	return RedisClient.Set(ctx, key, data, StatsCacheTTL).Err()
}

// Get URL stats from cache
func GetURLStats(shortCode string) (*models.StatsResponse, error) {
	if RedisClient == nil {
		return nil, redis.Nil
	}

	key := fmt.Sprintf(URLStatsKey, shortCode)
	data, err := RedisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var stats models.StatsResponse
	err = json.Unmarshal([]byte(data), &stats)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// Cache original URL mapping (to check if URL already exists)
func CacheOriginalURLMapping(originalURL string, shortCode string) error {
	if RedisClient == nil {
		return nil
	}

	key := fmt.Sprintf(OriginalURLKey, hashString(originalURL))
	return RedisClient.Set(ctx, key, shortCode, DefaultCacheTTL).Err()
}

// Get short code for original URL
func GetShortCodeForOriginalURL(originalURL string) (string, error) {
	if RedisClient == nil {
		return "", redis.Nil
	}

	key := fmt.Sprintf(OriginalURLKey, hashString(originalURL))
	return RedisClient.Get(ctx, key).Result()
}

// Increment click count in cache
func IncrementClickCount(shortCode string) error {
	if RedisClient == nil {
		return nil
	}

	key := fmt.Sprintf("url:clicks:%s", shortCode)
	return RedisClient.Incr(ctx, key).Err()
}

// Get click count from cache
func GetClickCount(shortCode string) (int64, error) {
	if RedisClient == nil {
		return 0, redis.Nil
	}

	key := fmt.Sprintf("url:clicks:%s", shortCode)
	return RedisClient.Get(ctx, key).Int64()
}

// Invalidate cache for a short code
func InvalidateCache(shortCode string) {
	if RedisClient == nil {
		return
	}

	keys := []string{
		fmt.Sprintf(URLMappingKey, shortCode),
		fmt.Sprintf(URLStatsKey, shortCode),
		fmt.Sprintf("url:clicks:%s", shortCode),
	}

	for _, key := range keys {
		RedisClient.Del(ctx, key)
	}
}

// Simple hash function for URL keys
func hashString(s string) string {
	hash := uint32(0)
	for _, c := range s {
		hash = hash*31 + uint32(c)
	}
	return fmt.Sprintf("%x", hash)
}

// Health check for Redis
func IsRedisHealthy() bool {
	if RedisClient == nil {
		return false
	}

	_, err := RedisClient.Ping(ctx).Result()
	return err == nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
