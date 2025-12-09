package campaign

import (
	"context"
	"time"
)

type TargetType string

const (
	TargetTypeAll     TargetType = "ALL"
	TargetTypeSegment TargetType = "SEGMENT"
)

type Campaign struct {
	ID            int64      `json:"id"`
	Title         string     `json:"title"`
	ImageURL      string     `json:"image_url"`
	ActionURL     string     `json:"action_url"`
	Priority      int        `json:"priority"`
	StartTime     time.Time  `json:"start_time"`
	EndTime       time.Time  `json:"end_time"`
	MaxFrequency  int        `json:"max_frequency"`
	TargetType    TargetType `json:"target_type"`
	TargetSegment string     `json:"target_segment,omitempty"` // If TargetType == SEGMENT
	IsActive      bool       `json:"is_active,omitempty"`      // For DB/Admin
}

// Repository (Redis - Hot Path)
type Repository interface {
	GetActiveCampaignIDs(ctx context.Context) ([]int64, error)
	GetCampaignsMetadata(ctx context.Context, ids []int64) (map[int64]*Campaign, error)
	IsUserTargeted(ctx context.Context, campaignID int64, userID int64) (bool, error)
	GetUserImpressions(ctx context.Context, userID int64, campaignIDs []int64) (map[int64]int, error)
	IncrementImpression(ctx context.Context, userID int64, campaignID int64) error
	
	// Write methods for Syncing/Admin
	SaveCampaign(ctx context.Context, c *Campaign) error
	RemoveCampaign(ctx context.Context, id int64) error
}

// Store (PostgreSQL - Persistence)
type Store interface {
	Create(ctx context.Context, c *Campaign) error
	Update(ctx context.Context, c *Campaign) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*Campaign, error)
	List(ctx context.Context) ([]*Campaign, error)
}
