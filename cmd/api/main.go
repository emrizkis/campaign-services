package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"campaign-management/internal/campaign"
	campaignPostgres "campaign-management/internal/platform/postgres"
	campaignRedis "campaign-management/internal/platform/redis"

	_ "campaign-management/docs" // Import generated docs

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @title           Campaign Service API
// @version         1.0
// @description     High-performance Campaign Popup Service with Redis & PostgreSQL.
// @host            localhost:8080
// @BasePath        /
func main() {
	// 1. Init Redis Connection (Infra)
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Could not initialize Redis: %v", err)
	}
	defer rdb.Close()
	log.Println("✅ Redis Connected")

	// 2. Init SQL Connection (Infra)
	// User: user (local), DB: campaign_db, SSL: disable
	connStr := "user=user dbname=campaign_db sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Could not open SQL connection: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Could not connect to PostgreSQL: %v", err)
	}
	log.Println("✅ PostgreSQL Connected (campaign_db)")

	// 3. Init Layers
	repo := campaignRedis.NewRepository(rdb)
	store := campaignPostgres.NewStore(db)
	svc := campaign.NewService(repo, store)
	handler := NewHandler(svc, rdb)

	// 4. Routes
	mux := http.NewServeMux()
	mux.HandleFunc("POST /debug/seed", handler.SeedData)
	mux.HandleFunc("GET /v1/campaigns/popup", handler.GetPopup)
	mux.HandleFunc("POST /v1/campaigns/impression", handler.RegisterImpression)
	mux.HandleFunc("POST /debug/sync", handler.SyncData)

	// Admin
	mux.HandleFunc("POST /admin/campaigns", handler.CreateCampaign)
	mux.HandleFunc("PUT /admin/campaigns", handler.UpdateCampaign)
	mux.HandleFunc("DELETE /admin/campaigns", handler.DeleteCampaign)
	mux.HandleFunc("GET /admin/campaigns", handler.ListCampaigns)
	mux.HandleFunc("GET /admin/campaigns/detail", handler.GetCampaign)

	// Swagger
	mux.HandleFunc("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	// 5. Start Server
	log.Println("🚀 Campaign Service listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
