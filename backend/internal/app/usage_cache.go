package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const usageLookupCacheTTL = 30 * time.Second

type usageCacheEntry[T any] struct {
	value      T
	expiresAt  time.Time
	generation uint64
}

type usageQueryCache struct {
	mu sync.Mutex

	prices           usageCacheEntry[map[[2]string]ModelPrice]
	priceGeneration  uint64
	users            usageCacheEntry[map[string]userInfo]
	userGeneration   uint64
	options          map[string]usageCacheEntry[map[string]any]
	optionGeneration uint64

	group singleflight.Group
}

func (a *App) cachedPriceMap(ctx context.Context) (map[[2]string]ModelPrice, error) {
	now := time.Now()
	a.usageCache.mu.Lock()
	entry := a.usageCache.prices
	generation := a.usageCache.priceGeneration
	if entry.value != nil && entry.generation == generation && now.Before(entry.expiresAt) {
		a.usageCache.mu.Unlock()
		return entry.value, nil
	}
	a.usageCache.mu.Unlock()

	value, err, _ := a.usageCache.group.Do("prices", func() (any, error) {
		prices, err := a.loadPriceMap(ctx)
		if err != nil {
			return nil, err
		}
		a.usageCache.mu.Lock()
		if a.usageCache.priceGeneration == generation {
			a.usageCache.prices = usageCacheEntry[map[[2]string]ModelPrice]{
				value:      prices,
				expiresAt:  time.Now().Add(usageLookupCacheTTL),
				generation: generation,
			}
		}
		a.usageCache.mu.Unlock()
		return prices, nil
	})
	if err != nil {
		return nil, err
	}
	return value.(map[[2]string]ModelPrice), nil
}

func (a *App) cachedAdminUserLookup(ctx context.Context) (map[string]userInfo, error) {
	now := time.Now()
	a.usageCache.mu.Lock()
	entry := a.usageCache.users
	generation := a.usageCache.userGeneration
	if entry.value != nil && entry.generation == generation && now.Before(entry.expiresAt) {
		a.usageCache.mu.Unlock()
		return entry.value, nil
	}
	a.usageCache.mu.Unlock()

	value, err, _ := a.usageCache.group.Do("users", func() (any, error) {
		users, err := a.loadAdminUserLookup(ctx)
		if err != nil {
			return nil, err
		}
		a.usageCache.mu.Lock()
		if a.usageCache.userGeneration == generation {
			a.usageCache.users = usageCacheEntry[map[string]userInfo]{
				value:      users,
				expiresAt:  time.Now().Add(usageLookupCacheTTL),
				generation: generation,
			}
		}
		a.usageCache.mu.Unlock()
		return users, nil
	})
	if err != nil {
		return nil, err
	}
	return value.(map[string]userInfo), nil
}

func (a *App) cachedUsageOptions(ctx context.Context, key string, loader func(context.Context) (map[string]any, error)) (map[string]any, error) {
	now := time.Now()
	a.usageCache.mu.Lock()
	entry, ok := a.usageCache.options[key]
	generation := a.usageCache.optionGeneration
	if ok && entry.generation == generation && now.Before(entry.expiresAt) {
		a.usageCache.mu.Unlock()
		return entry.value, nil
	}
	a.usageCache.mu.Unlock()

	groupKey := fmt.Sprintf("options:%d:%s", generation, key)
	value, err, _ := a.usageCache.group.Do(groupKey, func() (any, error) {
		options, err := loader(ctx)
		if err != nil {
			return nil, err
		}
		a.usageCache.mu.Lock()
		if a.usageCache.optionGeneration == generation {
			if a.usageCache.options == nil {
				a.usageCache.options = map[string]usageCacheEntry[map[string]any]{}
			}
			storedAt := time.Now()
			for cachedKey, cachedEntry := range a.usageCache.options {
				if !storedAt.Before(cachedEntry.expiresAt) {
					delete(a.usageCache.options, cachedKey)
				}
			}
			a.usageCache.options[key] = usageCacheEntry[map[string]any]{
				value:      options,
				expiresAt:  storedAt.Add(usageLookupCacheTTL),
				generation: generation,
			}
		}
		a.usageCache.mu.Unlock()
		return options, nil
	})
	if err != nil {
		return nil, err
	}
	return value.(map[string]any), nil
}

func (a *App) invalidateUsagePrices() {
	a.usageCache.mu.Lock()
	a.usageCache.priceGeneration++
	a.usageCache.prices = usageCacheEntry[map[[2]string]ModelPrice]{}
	a.usageCache.mu.Unlock()
	a.usageCache.group.Forget("prices")
}

func (a *App) invalidateUsageUsers() {
	a.usageCache.mu.Lock()
	a.usageCache.userGeneration++
	a.usageCache.users = usageCacheEntry[map[string]userInfo]{}
	a.usageCache.optionGeneration++
	a.usageCache.options = nil
	a.usageCache.mu.Unlock()
	a.usageCache.group.Forget("users")
}

func (a *App) invalidateUsageOptions() {
	a.usageCache.mu.Lock()
	a.usageCache.optionGeneration++
	a.usageCache.options = nil
	a.usageCache.mu.Unlock()
}
