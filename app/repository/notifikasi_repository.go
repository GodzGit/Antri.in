package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/GodzGit/Antri.in/app/model"
)

var ErrIDTidakValid = errors.New("id tidak valid")

type notifikasiDocument struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	UserID    int64         `bson:"user_id"`
	TiketID   int64         `bson:"tiket_id"`
	Judul     string        `bson:"judul"`
	Pesan     string        `bson:"pesan"`
	Dibaca    bool          `bson:"dibaca"`
	EventKey  string        `bson:"event_key,omitempty"`
	CreatedAt time.Time     `bson:"created_at"`
}

func (d notifikasiDocument) toModel() model.Notifikasi {
	return model.Notifikasi{
		ID:        d.ID.Hex(),
		UserID:    d.UserID,
		TiketID:   d.TiketID,
		Judul:     d.Judul,
		Pesan:     d.Pesan,
		Dibaca:    d.Dibaca,
		CreatedAt: d.CreatedAt,
	}
}

type NotifikasiRepository struct {
	col *mongo.Collection
}

func NewNotifikasiRepository(db *mongo.Database) *NotifikasiRepository {
	return &NotifikasiRepository{col: db.Collection("notifikasi")}
}

func (r *NotifikasiRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "_id", Value: -1}}},
		{
			Keys: bson.D{{Key: "event_key", Value: 1}},
			Options: options.Index().SetUnique(true).
				SetPartialFilterExpression(bson.D{{Key: "event_key", Value: bson.D{{Key: "$exists", Value: true}}}}),
		},
	})
	if err != nil {
		return fmt.Errorf("membuat index notifikasi: %w", err)
	}
	return nil
}

func (r *NotifikasiRepository) Create(ctx context.Context, n *model.Notifikasi) error {
	_, err := r.insert(ctx, "", n)
	return err
}

func (r *NotifikasiRepository) CreateSekali(ctx context.Context, eventKey string, n *model.Notifikasi) (bool, error) {
	return r.insert(ctx, eventKey, n)
}

func (r *NotifikasiRepository) insert(ctx context.Context, eventKey string, n *model.Notifikasi) (bool, error) {
	doc := notifikasiDocument{
		UserID:    n.UserID,
		TiketID:   n.TiketID,
		Judul:     n.Judul,
		Pesan:     n.Pesan,
		EventKey:  eventKey,
		CreatedAt: time.Now().UTC(),
	}

	res, err := r.col.InsertOne(ctx, doc)
	if eventKey != "" && mongo.IsDuplicateKeyError(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("menyimpan notifikasi: %w", err)
	}

	doc.ID = res.InsertedID.(bson.ObjectID)
	*n = doc.toModel()
	return true, nil
}

func (r *NotifikasiRepository) ListByUser(ctx context.Context, userID int64, cursor string, limit int) ([]model.Notifikasi, string, error) {
	filter := bson.D{{Key: "user_id", Value: userID}}
	if cursor != "" {
		after, err := bson.ObjectIDFromHex(cursor)
		if err != nil {
			return nil, "", ErrIDTidakValid
		}
		filter = append(filter, bson.E{Key: "_id", Value: bson.D{{Key: "$lt", Value: after}}})
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "_id", Value: -1}}).
		SetLimit(int64(limit + 1))

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, "", fmt.Errorf("mengambil notifikasi: %w", err)
	}

	var docs []notifikasiDocument
	if err := cur.All(ctx, &docs); err != nil {
		return nil, "", fmt.Errorf("membaca notifikasi: %w", err)
	}

	next := ""
	if len(docs) > limit {
		docs = docs[:limit]
		next = docs[limit-1].ID.Hex()
	}

	items := make([]model.Notifikasi, 0, len(docs))
	for _, d := range docs {
		items = append(items, d.toModel())
	}
	return items, next, nil
}

func (r *NotifikasiRepository) MarkRead(ctx context.Context, id string, userID int64) (bool, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return false, ErrIDTidakValid
	}

	res, err := r.col.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: oid}, {Key: "user_id", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "dibaca", Value: true}}}})
	if err != nil {
		return false, fmt.Errorf("menandai notifikasi: %w", err)
	}
	return res.MatchedCount > 0, nil
}
