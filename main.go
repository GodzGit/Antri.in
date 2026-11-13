package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/repository"
	"github.com/GodzGit/Antri.in/app/service"
	"github.com/GodzGit/Antri.in/config"
	"github.com/GodzGit/Antri.in/database"
	"github.com/GodzGit/Antri.in/middleware"
	"github.com/GodzGit/Antri.in/route"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	// ===== PostgreSQL =====
	pgPool, err := database.ConnectPostgres(ctx, cfg.DB)
	if err != nil {
		log.Fatalf("gagal koneksi PostgreSQL: %v", err)
	}
	log.Println("PostgreSQL terhubung")
	defer pgPool.Close()

	// ===== Seeder Role dan Permission =====
	if err := database.SeedRolesAndPermissions(ctx, pgPool); err != nil {
		log.Fatalf("seeder role dan permission gagal: %v", err)
	}

	// ===== Repository Auth dan User =====
	userRepo := repository.NewUserRepository(pgPool)
	roleRepo := repository.NewRoleRepository(pgPool)
	tokenRepo := repository.NewTokenRepository(pgPool)
	txManager := repository.NewTxManager(pgPool)

	// ===== Service Auth dan User =====
	authSvc := service.NewAuthService(
		userRepo,
		roleRepo,
		tokenRepo,
		txManager,
		cfg.JWT,
	)

	userSvc := service.NewUserService(
		userRepo,
		roleRepo,
	)

	// ===== MongoDB (non-fatal dulu) =====
	mongoClient, err := database.ConnectMongo(ctx, cfg.Mongo)
	if err != nil {
		log.Printf("MongoDB tidak terhubung: %v", err)
	} else {
		log.Println("MongoDB terhubung")
		defer func() {
			_ = mongoClient.Disconnect(ctx)
		}()
	}

	// ===== Redis (non-fatal dulu) =====
	redisClient, err := database.ConnectRedis(ctx, cfg.Redis)
	if err != nil {
		log.Printf("Redis tidak terhubung: %v", err)
	} else {
		log.Println("Redis terhubung")
		defer redisClient.Close()
	}

	// ===== Fiber App =====
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler(),
	})

	app.Use(middleware.Logger())

	// ===== Repository dan Service Layanan =====
	layananRepo := repository.NewLayananRepository(pgPool)
	layananService := service.NewLayananService(layananRepo)

	// ===== Repository dan Service Loket =====
	loketRepo := repository.NewLoketRepository(pgPool)
	loketService := service.NewLoketService(loketRepo)

	// ===== Route Dependencies =====
	deps := route.Deps{
		Auth:    authSvc,
		User:    userSvc,
		Layanan: layananService,
		Loket:   loketService,
	}

	// ===== Daftarkan Route =====
	route.Setup(app, deps, cfg.JWT.Secret)

	// ===== Jalankan Server =====
	addr := ":" + cfg.App.Port
	log.Printf("ANTRI.IN API berjalan di http://localhost%s", addr)

	if err := app.Listen(addr); err != nil {
		log.Fatalf("gagal menjalankan server: %v", err)
	}
}
