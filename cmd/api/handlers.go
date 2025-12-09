package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"campaign-management/internal/campaign"

	"github.com/redis/go-redis/v9"
)

type Handler struct {
	service *campaign.Service
	rdb     *redis.Client // Needed for seeding
}

func NewHandler(service *campaign.Service, rdb *redis.Client) *Handler {
	return &Handler{service: service, rdb: rdb}
}

// GetPopup godoc
// @Summary      Get Popup for User
// @Description  Determines the best campaign popup for a user based on priority, time, and targeting.
// @Tags         Client
// @Accept       json
// @Produce      json
// @Param        user_id   query      int  true  "User ID"
// @Success      200  {object}  campaign.Campaign
// @Success      204  "No Content (No suitable campaign)"
// @Failure      400  {string}  string "Invalid User ID"
// @Router       /v1/campaigns/popup [get]
func (h *Handler) GetPopup(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Calculate latency
	start := time.Now()
	
	p, err := h.service.GetPopup(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	duration := time.Since(start)
	w.Header().Set("X-Response-Time", duration.String())
	w.Header().Set("Content-Type", "application/json")

	if p == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	json.NewEncoder(w).Encode(p)
}

type ImpressionRequest struct {
	UserID     int64 `json:"user_id"`
	CampaignID int64 `json:"campaign_id"`
}

// RegisterImpression godoc
// @Summary      Track Impression
// @Description  Records that a user has seen a campaign.
// @Tags         Client
// @Accept       json
// @Produce      json
// @Param        request body ImpressionRequest true "Impression Request"
// @Success      200  "OK"
// @Router       /v1/campaigns/impression [post]
func (h *Handler) RegisterImpression(w http.ResponseWriter, r *http.Request) {
	var req ImpressionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.RegisterImpression(r.Context(), req.UserID, req.CampaignID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// POST /debug/seed
// Helper to inject data into Redis for testing
func (h *Handler) SeedData(w http.ResponseWriter, r *http.Request) {
	// ... (Seed logic)
	// For simplicity, returning just generic message since we use DB seeding now
	// But let's keep it compatible if needed, or just deprecate it
	fmt.Fprintln(w, "Deprecated: Use /admin/campaigns or DB Seeding + /debug/sync")
}

// --- Admin Handlers ---

// CreateCampaign godoc
// @Summary      Create New Campaign
// @Description  Creates a campaign in DB and syncs to Redis.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        campaign body campaign.Campaign true "Campaign Data"
// @Success      201  {object}  campaign.Campaign
// @Router       /admin/campaigns [post]
func (h *Handler) CreateCampaign(w http.ResponseWriter, r *http.Request) {
	var c campaign.Campaign
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := h.service.CreateCampaign(r.Context(), &c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}

// UpdateCampaign godoc
// @Summary      Update Campaign
// @Description  Updates a campaign in DB and syncs to Redis.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        campaign body campaign.Campaign true "Campaign Data"
// @Success      200  {object}  campaign.Campaign
// @Router       /admin/campaigns [put]
func (h *Handler) UpdateCampaign(w http.ResponseWriter, r *http.Request) {
	var c campaign.Campaign
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if c.ID == 0 {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateCampaign(r.Context(), &c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	json.NewEncoder(w).Encode(c)
}

// DeleteCampaign godoc
// @Summary      Delete Campaign
// @Description  Deletes a campaign from DB and Redis.
// @Tags         Admin
// @Param        id   query      int  true  "Campaign ID"
// @Success      204  "No Content"
// @Router       /admin/campaigns [delete]
func (h *Handler) DeleteCampaign(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteCampaign(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListCampaigns godoc
// @Summary      List All Campaigns
// @Description  Fetches all campaigns from PostgreSQL.
// @Tags         Admin
// @Produce      json
// @Success      200  {array}  campaign.Campaign
// @Router       /admin/campaigns [get]
func (h *Handler) ListCampaigns(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.ListCampaigns(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(list)
}

// GetCampaign godoc
// @Summary      Get Campaign Detail
// @Description  Fetches a single campaign from PostgreSQL.
// @Tags         Admin
// @Produce      json
// @Param        id   query      int  true  "Campaign ID"
// @Success      200  {object}  campaign.Campaign
// @Router       /admin/campaigns/detail [get]
func (h *Handler) GetCampaign(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	c, err := h.service.GetCampaign(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if c == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(c)
}

// SyncData godoc
// @Summary      Sync DB to Redis
// @Description  Manually triggers synchronization of all campaigns from DB to Redis.
// @Tags         Debug
// @Success      200  "Synced"
// @Router       /debug/sync [post]
func (h *Handler) SyncData(w http.ResponseWriter, r *http.Request) {
	if err := h.service.SyncCampaigns(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Synced DB to Redis"))
}
