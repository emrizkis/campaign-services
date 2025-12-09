package campaign

import (
	"context"
	"time"
)

type Service struct {
	repo  Repository // Redis
	store Store      // Postgres
}

func NewService(repo Repository, store Store) *Service {
	return &Service{repo: repo, store: store}
}

// GetPopup determines which popup to show for a user.
func (s *Service) GetPopup(ctx context.Context, userID int64) (*Campaign, error) {
	// 1. Get ALL active campaign IDs (Already Sorted by Priority in Redis ZSET)
	activeIDs, err := s.repo.GetActiveCampaignIDs(ctx)
	if err != nil {
		return nil, err // Or fail silent returning nil
	}
	if len(activeIDs) == 0 {
		return nil, nil // No active campaigns
	}

	// 2. Fetch Metadata for ALL candidates (Pipeline)
	campMap, err := s.repo.GetCampaignsMetadata(ctx, activeIDs)
	if err != nil {
		return nil, err
	}

	// 3. Evaluation Loop (Highest Priority First)
	now := time.Now()

	for _, id := range activeIDs {
		camp, exists := campMap[id]
		if !exists {
			continue
		}

		// A. Time Check
		if now.Before(camp.StartTime) || now.After(camp.EndTime) {
			continue
		}

		// B. Target Check
		isTargeted := true
		if camp.TargetType == TargetTypeSegment {
			// Check Whitelist/Segment
			isTargeted, err = s.repo.IsUserTargeted(ctx, id, userID)
			if err != nil {
				// Log error, skip this campaign
				continue
			}
		}

		if !isTargeted {
			continue
		}

		// C. Frequency Cap Check
		impressionsMap, err := s.repo.GetUserImpressions(ctx, userID, []int64{id})
		if err != nil {
			continue
		}
		
		seenCount := impressionsMap[id]
		if seenCount >= camp.MaxFrequency {
			continue // Cap reached
		}

		// D. Winner Found!
		return camp, nil
	}

	return nil, nil // No eligible campaign found
}

func (s *Service) RegisterImpression(ctx context.Context, userID int64, campaignID int64) error {
	return s.repo.IncrementImpression(ctx, userID, campaignID)
}

// --- CRUD / Admin ---

func (s *Service) CreateCampaign(ctx context.Context, c *Campaign) error {
	// 1. Save to DB (Single Source of Truth)
	if err := s.store.Create(ctx, c); err != nil {
		return err
	}
	// 2. Sync to Redis (Cache / Hot Path)
	return s.repo.SaveCampaign(ctx, c)
}

func (s *Service) UpdateCampaign(ctx context.Context, c *Campaign) error {
	// 1. Update DB
	if err := s.store.Update(ctx, c); err != nil {
		return err
	}
	// 2. Sync Redis
	return s.repo.SaveCampaign(ctx, c)
}

func (s *Service) DeleteCampaign(ctx context.Context, id int64) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}
	return s.repo.RemoveCampaign(ctx, id)
}

func (s *Service) ListCampaigns(ctx context.Context) ([]*Campaign, error) {
	return s.store.List(ctx) // DB Only
}

func (s *Service) GetCampaign(ctx context.Context, id int64) (*Campaign, error) {
	return s.store.GetByID(ctx, id) // DB Only
}

func (s *Service) SyncCampaigns(ctx context.Context) error {
	// 1. Fetch All from DB
	list, err := s.store.List(ctx)
	if err != nil {
		return err
	}
	
	// 2. Push to Redis
	for _, c := range list {
		if err := s.repo.SaveCampaign(ctx, c); err != nil {
			return err
		}
	}
	return nil
}
