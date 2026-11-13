package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/GodzGit/Antri.in/app/repository"
	"github.com/GodzGit/Antri.in/config"
)

type Deps struct {
	Pool   *pgxpool.Pool
	Mongo  *mongo.Database
	Redis  config.RedisConfig
	Logger *slog.Logger
}

func Start(ctx context.Context, d Deps) (func(), error) {
	riwayatRepo := repository.NewRiwayatRepository(d.Mongo)
	notifikasiRepo := repository.NewNotifikasiRepository(d.Mongo)
	if err := riwayatRepo.EnsureIndexes(ctx); err != nil {
		return nil, err
	}
	if err := notifikasiRepo.EnsureIndexes(ctx); err != nil {
		return nil, err
	}

	redisOpt := asynq.RedisClientOpt{Addr: d.Redis.Addr, Password: d.Redis.Password, DB: d.Redis.DB}
	client := asynq.NewClient(redisOpt)

	relayCtx, stopRelay := context.WithCancel(ctx)
	relay := NewRelay(repository.NewOutboxRepository(d.Pool), repository.NewTxManager(d.Pool),
		client, d.Logger, 2*time.Second, 50)
	go relay.Run(relayCtx)

	handler := NewHandler(riwayatRepo, notifikasiRepo, repository.NewInfoAntreanRepository(d.Pool), d.Logger)
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskOutboxTiket, handler.ProcessTask)

	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 5,
		Queues:      map[string]int{QueueTiket: 1},
	})
	if err := srv.Start(mux); err != nil {
		stopRelay()
		_ = client.Close()
		return nil, fmt.Errorf("menjalankan worker: %w", err)
	}

	return func() {
		stopRelay()
		srv.Shutdown()
		_ = client.Close()
	}, nil
}
