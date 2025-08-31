package services

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// LockService provides distributed locking using Redis
type LockService struct {
	client *redis.Client
}

// NewLockService creates a new lock service
func NewLockService(client *redis.Client) *LockService {
	return &LockService{client: client}
}

// AcquireLock acquires a distributed lock with expiration
func (s *LockService) AcquireLock(ctx context.Context, lockID string, expiration time.Duration) (bool, error) {
	key := fmt.Sprintf("lock:%s", lockID)
	
	// Use SET with NX (only if not exists) and EX (expiration)
	result, err := s.client.SetNX(ctx, key, "1", expiration).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}
	
	return result, nil
}

// ReleaseLock releases a distributed lock
func (s *LockService) ReleaseLock(ctx context.Context, lockID string) error {
	key := fmt.Sprintf("lock:%s", lockID)
	
	_, err := s.client.Del(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	
	return nil
}

// ExtendLock extends the expiration of an existing lock
func (s *LockService) ExtendLock(ctx context.Context, lockID string, expiration time.Duration) (bool, error) {
	key := fmt.Sprintf("lock:%s", lockID)
	
	// Check if lock exists and extend it
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check lock existence: %w", err)
	}
	
	if exists == 0 {
		return false, nil // Lock doesn't exist
	}
	
	// Extend the lock
	_, err = s.client.Expire(ctx, key, expiration).Result()
	if err != nil {
		return false, fmt.Errorf("failed to extend lock: %w", err)
	}
	
	return true, nil
}

// IsLocked checks if a lock is currently held
func (s *LockService) IsLocked(ctx context.Context, lockID string) (bool, error) {
	key := fmt.Sprintf("lock:%s", lockID)
	
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check lock: %w", err)
	}
	
	return exists > 0, nil
}

// WithLock executes a function while holding a distributed lock
func (s *LockService) WithLock(ctx context.Context, lockID string, expiration time.Duration, fn func() error) error {
	// Try to acquire the lock
	acquired, err := s.AcquireLock(ctx, lockID, expiration)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	
	if !acquired {
		return fmt.Errorf("could not acquire lock: %s", lockID)
	}
	
	// Ensure lock is released
	defer func() {
		if releaseErr := s.ReleaseLock(ctx, lockID); releaseErr != nil {
			// Log the error but don't override the original error
			fmt.Printf("Warning: failed to release lock %s: %v\n", lockID, releaseErr)
		}
	}()
	
	// Execute the function
	return fn()
}

// TryWithLock tries to execute a function with a lock, returns immediately if lock cannot be acquired
func (s *LockService) TryWithLock(ctx context.Context, lockID string, expiration time.Duration, fn func() error) (bool, error) {
	// Try to acquire the lock
	acquired, err := s.AcquireLock(ctx, lockID, expiration)
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}
	
	if !acquired {
		return false, nil // Could not acquire lock, but no error
	}
	
	// Ensure lock is released
	defer func() {
		if releaseErr := s.ReleaseLock(ctx, lockID); releaseErr != nil {
			// Log the error but don't override the original error
			fmt.Printf("Warning: failed to release lock %s: %v\n", lockID, releaseErr)
		}
	}()
	
	// Execute the function
	err = fn()
	return true, err
}

// GetLockTTL returns the remaining time-to-live of a lock
func (s *LockService) GetLockTTL(ctx context.Context, lockID string) (time.Duration, error) {
	key := fmt.Sprintf("lock:%s", lockID)
	
	ttl, err := s.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get lock TTL: %w", err)
	}
	
	return ttl, nil
}