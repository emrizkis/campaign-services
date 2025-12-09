-- Campaigns Table
CREATE TABLE IF NOT EXISTS campaigns (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    image_url TEXT,
    action_url TEXT,
    priority INT DEFAULT 0,
    start_time TIMESTAMP WITH TIME ZONE,
    end_time TIMESTAMP WITH TIME ZONE,
    max_frequency INT DEFAULT 1,
    target_type VARCHAR(20) DEFAULT 'ALL', -- 'ALL', 'SEGMENT'
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Campaign Targets (Whitelisting specific users)
CREATE TABLE IF NOT EXISTS campaign_targets (
    campaign_id BIGINT REFERENCES campaigns(id),
    user_id BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (campaign_id, user_id)
);

-- Campaign Impressions (Analytics)
CREATE TABLE IF NOT EXISTS campaign_impressions (
    id BIGSERIAL PRIMARY KEY,
    campaign_id BIGINT REFERENCES campaigns(id),
    user_id BIGINT,
    action VARCHAR(50), -- 'VIEW', 'CLICK', 'DISMISS'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for analytics speed
CREATE INDEX IF NOT EXISTS idx_impressions_campaign_user ON campaign_impressions(campaign_id, user_id);
