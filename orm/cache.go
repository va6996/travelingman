package orm

import (
	"time"

	"gorm.io/gorm"
)

// APICache stores cached API responses
type APICache struct {
	Key       string `gorm:"primaryKey"`
	Value     []byte `gorm:"type:bytea"` // Compressed/Raw JSON
	CreatedAt time.Time
	ExpiresAt time.Time `gorm:"index"`
}

// GetCacheEntry retrieves a valid cache entry
func GetCacheEntry(db *gorm.DB, key string) (*APICache, error) {
	var entry APICache
	// Check for existence and expiry
	err := db.Where("key = ? AND expires_at > ?", key, time.Now()).First(&entry).Error
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// SetCacheEntry upserts a cache entry
func SetCacheEntry(db *gorm.DB, key string, value []byte, ttl time.Duration) error {
	entry := APICache{
		Key:       key,
		Value:     value,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}
	// Upsert (On Conflict Do Update)
	return db.Save(&entry).Error
}

// CleanupCache removes expired entries
func CleanupCache(db *gorm.DB) error {
	return db.Where("expires_at < ?", time.Now()).Delete(&APICache{}).Error
}
