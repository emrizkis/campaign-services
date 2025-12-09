package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"campaign-management/internal/campaign"

	"github.com/redis/go-redis/v9"
)

type Repository struct {
	rdb *redis.Client
}

func NewRepository(rdb *redis.Client) *Repository {
	return &Repository{rdb: rdb}
}

// GetActiveCampaignIDs fetches all active campaign IDs sorted by priority (ZSET).
func (r *Repository) GetActiveCampaignIDs(ctx context.Context) ([]int64, error) {
	// Key: campaigns:active (ZSET score=priority, member=id)
	// ZREVRANGE 0 -1 to get all, highest priority first
	idsStr, err := r.rdb.ZRevRange(ctx, "campaigns:active", 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get active campaigns: %w", err)
	}

	ids := make([]int64, len(idsStr))
	for i, s := range idsStr {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			continue // Skip invalid IDs
		}
		ids[i] = id
	}
	return ids, nil
}

// GetCampaignsMetadata fetches metadata for multiple campaigns (Pipeline).
func (r *Repository) GetCampaignsMetadata(ctx context.Context, ids []int64) (map[int64]*campaign.Campaign, error) {
	pipe := r.rdb.Pipeline()
	cmds := make(map[int64]*redis.StringCmd)

	for _, id := range ids {
		key := fmt.Sprintf("campaign:%d:meta", id)
		cmds[id] = pipe.Get(ctx, key) // Assuming JSON stored in string key for simplicity
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to exec pipeline metadata: %w", err)
	}

	result := make(map[int64]*campaign.Campaign)
	for id, cmd := range cmds {
		val, err := cmd.Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			continue
		}

		var c campaign.Campaign
		if err := json.Unmarshal([]byte(val), &c); err == nil {
			c.ID = id
			result[id] = &c
		}
	}
	return result, nil
}

// IsUserTargeted checks if a user is in the target list (BITMAP).
func (r *Repository) IsUserTargeted(ctx context.Context, campaignID int64, userID int64) (bool, error) {
	key := fmt.Sprintf("campaign:%d:users", campaignID)
	// GETBIT returns 0 or 1
	bit, err := r.rdb.GetBit(ctx, key, int64(userID)).Result()
	if err != nil {
		return false, err
	}
	return bit == 1, nil
}

// GetUserImpressions fetches how many times a user has seen specific campaigns.
func (r *Repository) GetUserImpressions(ctx context.Context, userID int64, campaignIDs []int64) (map[int64]int, error) {
	key := fmt.Sprintf("user:%d:impressions", userID)
	
	// HMGET
	fields := make([]string, len(campaignIDs))
	for i, id := range campaignIDs {
		fields[i] = strconv.FormatInt(id, 10)
	}

	vals, err := r.rdb.HMGet(ctx, key, fields...).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[int64]int)
	for i, val := range vals {
		if val == nil {
			result[campaignIDs[i]] = 0
			continue
		}
		
		// Redis returns string or int depending on client version, handle safe conversion
		switch v := val.(type) {
		case string:
			count, _ := strconv.Atoi(v)
			result[campaignIDs[i]] = count
		case int64: // if redis client auto-parses
			result[campaignIDs[i]] = int(v)
		default:
			result[campaignIDs[i]] = 0
		}
	}
	return result, nil
}

func (r *Repository) IncrementImpression(ctx context.Context, userID int64, campaignID int64) error {
	key := fmt.Sprintf("user:%d:impressions", userID)
	// Increment by 1
	if err := r.rdb.HIncrBy(ctx, key, strconv.FormatInt(campaignID, 10), 1).Err(); err != nil {
		return err
	}
	// Set TTL to 24h just in case, ensuring it doesn't grow forever if logic changes
	r.rdb.Expire(ctx, key, 24*time.Hour)
	return nil
}

// SaveCampaign syncs metadata to Redis and updates "Active" list if needed.
func (r *Repository) SaveCampaign(ctx context.Context, c *campaign.Campaign) error {
	pipe := r.rdb.Pipeline()

	// 1. Save Meta (Always)
	bytes, _ := json.Marshal(c)
	key := fmt.Sprintf("campaign:%d:meta", c.ID)
	pipe.Set(ctx, key, bytes, 0) // No TTL for now, or match campaign end time

	// 2. Manage Active List
	activeKey := "campaigns:active"
	if c.IsActive {
		// Add/Update score
		pipe.ZAdd(ctx, activeKey, redis.Z{Score: float64(c.Priority), Member: c.ID})
	} else {
		// Remove if inactive
		pipe.ZRem(ctx, activeKey, c.ID)
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (r *Repository) RemoveCampaign(ctx context.Context, id int64) error {
	pipe := r.rdb.Pipeline()
	pipe.ZRem(ctx, "campaigns:active", id)
	metaKey := fmt.Sprintf("campaign:%d:meta", id)
	pipe.Del(ctx, metaKey)
	// Set TTL to 24h just in case, ensuring it doesn't grow forever if logic changes
	// Note: Expire on a key that is being deleted in the same pipeline is redundant.
	// If the intention is to expire the key *after* deletion, it won't work.
	// If the intention is to expire it *instead* of deleting, then Del should be removed.
	// Assuming the intent is to ensure the key is eventually cleaned up if Del fails or is removed later.
	// For now, adding it to the pipeline for the metaKey.
	pipe.Expire(ctx, metaKey, 24*time.Hour)
	// Optional: Del other keys like targeting bit map if needed
	_, err := pipe.Exec(ctx)
	return err
}
