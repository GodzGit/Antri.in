package main

import (
	"context"
	"log"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/GodzGit/Antri.in/app/repository"
	"github.com/GodzGit/Antri.in/app/service"
	"github.com/GodzGit/Antri.in/config"
	"github.com/GodzGit/Antri.in/database"
	"github.com/GodzGit/Antri.in/middleware"
	"github.com/GodzGit/Antri.in/route"
	"github.com/GodzGit/Antri.in/worker"
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
	mongoSiap := err == nil
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
	redisSiap := err == nil
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

	tiketRepo := repository.NewTiketRepository(pgPool)
	outboxRepo := repository.NewOutboxRepository(pgPool)
	tiketService := service.NewTiketService(tiketRepo, layananRepo, outboxRepo, txManager)

	deps := route.Deps{
		Auth:    authSvc,
		User:    userSvc,
		Layanan: layananService,
		Loket:   loketService,
		Tiket:   tiketService,
	}

	// ===== Notifikasi, Estimasi, Riwayat, Statistik, dan Worker =====
	infoAntreanRepo := repository.NewInfoAntreanRepository(pgPool)
	deps.Estimasi = service.NewEstimasiService(infoAntreanRepo)
	if mongoSiap {
		mongoDB := mongoClient.Database(cfg.Mongo.DB)
		notifikasiRepo := repository.NewNotifikasiRepository(mongoDB)
		riwayatRepo := repository.NewRiwayatRepository(mongoDB)
		if err := notifikasiRepo.EnsureIndexes(ctx); err != nil {
			log.Fatalf("gagal membuat index notifikasi: %v", err)
		}
		if err := riwayatRepo.EnsureIndexes(ctx); err != nil {
			log.Fatalf("gagal membuat index riwayat: %v", err)
		}
		deps.Notifikasi = service.NewNotifikasiService(notifikasiRepo)
		deps.Riwayat = service.NewRiwayatService(infoAntreanRepo, riwayatRepo)
		deps.Statistik = service.NewStatistikService(infoAntreanRepo, riwayatRepo)

		if redisSiap {
			stopWorker, err := worker.Start(ctx, worker.Deps{
				Pool:   pgPool,
				Mongo:  mongoDB,
				Redis:  cfg.Redis,
				Logger: slog.Default(),
			})
			if err != nil {
				log.Fatalf("gagal menjalankan worker: %v", err)
			}
			log.Println("Worker outbox berjalan")
			defer stopWorker()
		}
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
