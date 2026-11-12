package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/config"
	"github.com/GodzGit/Antri.in/database"
	"github.com/GodzGit/Antri.in/middleware"
	"github.com/GodzGit/Antri.in/route"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	// ===== PostgreSQL (wajib untuk transaksi nanti) =====
	pgPool, err := database.ConnectPostgres(ctx, cfg.DB)
	if err != nil {
		log.Fatalf("gagal koneksi PostgreSQL: %v", err)
	}
	log.Println("✅ PostgreSQL terhubung")
	defer pgPool.Close()

	// ===== MongoDB (non-fatal dulu, supaya gampang dev) =====
	mongoClient, err := database.ConnectMongo(ctx, cfg.Mongo)
	if err != nil {
		log.Printf("⚠️  MongoDB tidak terhubung: %v", err)
	} else {
		log.Println("✅ MongoDB terhubung")
		defer func() {
			_ = mongoClient.Disconnect(ctx)
		}()
	}

	// ===== Redis (non-fatal dulu) =====
	redisClient, err := database.ConnectRedis(ctx, cfg.Redis)
	if err != nil {
		log.Printf("⚠️  Redis tidak terhubung: %v", err)
	} else {
		log.Println("✅ Redis terhubung")
		defer redisClient.Close()
	}

	// ===== Fiber app =====
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler(),
	})

	// ===== Daftarkan route =====
	route.Setup(app)

	// ===== Jalankan server =====
	addr := ":" + cfg.App.Port
	log.Printf("🚀 ANTRI.IN API berjalan di http://localhost%s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("gagal menjalankan server: %v", err)
	}
}