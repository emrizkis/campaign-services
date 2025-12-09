package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"campaign-management/internal/campaign"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Create(ctx context.Context, c *campaign.Campaign) error {
	query := `
		INSERT INTO campaigns (title, image_url, action_url, priority, start_time, end_time, max_frequency, target_type, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	err := s.db.QueryRowContext(ctx, query,
		c.Title, c.ImageURL, c.ActionURL, c.Priority, c.StartTime, c.EndTime, c.MaxFrequency, c.TargetType, c.IsActive,
	).Scan(&c.ID)
	
	if err != nil {
		return fmt.Errorf("failed to create campaign: %w", err)
	}
	return nil
}

func (s *Store) Update(ctx context.Context, c *campaign.Campaign) error {
	query := `
		UPDATE campaigns 
		SET title=$1, image_url=$2, action_url=$3, priority=$4, start_time=$5, end_time=$6, max_frequency=$7, target_type=$8, is_active=$9
		WHERE id=$10
	`
	res, err := s.db.ExecContext(ctx, query,
		c.Title, c.ImageURL, c.ActionURL, c.Priority, c.StartTime, c.EndTime, c.MaxFrequency, c.TargetType, c.IsActive, c.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update campaign: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("campaign not found")
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	// Hard delete for simplicity
	query := `DELETE FROM campaigns WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func (s *Store) GetByID(ctx context.Context, id int64) (*campaign.Campaign, error) {
	query := `
		SELECT id, title, COALESCE(image_url, ''), COALESCE(action_url, ''), priority, start_time, end_time, max_frequency, target_type, is_active
		FROM campaigns WHERE id = $1
	`
	c := &campaign.Campaign{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.Title, &c.ImageURL, &c.ActionURL, &c.Priority, &c.StartTime, &c.EndTime, &c.MaxFrequency, &c.TargetType, &c.IsActive,
	)
	if err == sql.ErrNoRows {
		return nil, nil // Return nil if not found
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Store) List(ctx context.Context) ([]*campaign.Campaign, error) {
	query := `
		SELECT id, title, COALESCE(image_url, ''), COALESCE(action_url, ''), priority, start_time, end_time, max_frequency, target_type, is_active
		FROM campaigns ORDER BY id DESC LIMIT 100
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*campaign.Campaign
	for rows.Next() {
		c := &campaign.Campaign{}
		if err := rows.Scan(
			&c.ID, &c.Title, &c.ImageURL, &c.ActionURL, &c.Priority, &c.StartTime, &c.EndTime, &c.MaxFrequency, &c.TargetType, &c.IsActive,
		); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}
