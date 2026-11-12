package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/GodzGit/Antri.in/config"
)

// ConnectMongo membuka koneksi ke MongoDB dan mengembalikan *mongo.Client.
// Client ini dipakai untuk mendapatkan *mongo.Database lewat client.Database(cfg.DB).
func ConnectMongo(ctx context.Context, cfg config.MongoConfig) (*mongo.Client, error) {
	clientOpts := options.Client().
		ApplyURI(cfg.URI).
		SetConnectTimeout(5 * time.Second)

	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}

	// Ping untuk memastikan benar-benar terhubung.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("ping mongo: %w", err)
	}

	return client, nil
}