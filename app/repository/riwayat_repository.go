package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/GodzGit/Antri.in/app/model"
)

type riwayatDocument struct {
	ID         bson.ObjectID `bson:"_id,omitempty"`
	OutboxID   int64         `bson:"outbox_id"`
	TiketID    int64         `bson:"tiket_id"`
	LayananID  int64         `bson:"layanan_id"`
	UserID     int64         `bson:"user_id"`
	LoketID    *int64        `bson:"loket_id,omitempty"`
	StatusLama string        `bson:"status_lama"`
	StatusBaru string        `bson:"status_baru"`
	Waktu      time.Time     `bson:"waktu"`
}

func (d riwayatDocument) toModel() model.RiwayatTiket {
	return model.RiwayatTiket{
		ID:         d.ID.Hex(),
		TiketID:    d.TiketID,
		LayananID:  d.LayananID,
		UserID:     d.UserID,
		LoketID:    d.LoketID,
		StatusLama: d.StatusLama,
		StatusBaru: d.StatusBaru,
		Waktu:      d.Waktu,
	}
}

type RiwayatRepository struct {
	col *mongo.Collection
}

func NewRiwayatRepository(db *mongo.Database) *RiwayatRepository {
	return &RiwayatRepository{col: db.Collection("riwayat_tiket")}
}

func (r *RiwayatRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "outbox_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "tiket_id", Value: 1}, {Key: "waktu", Value: 1}}},
		{Keys: bson.D{{Key: "layanan_id", Value: 1}, {Key: "waktu", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("membuat index riwayat: %w", err)
	}
	return nil
}

func (r *RiwayatRepository) CatatSekali(ctx context.Context, outboxID int64, rw model.RiwayatTiket) (bool, error) {
	_, err := r.col.InsertOne(ctx, riwayatDocument{
		OutboxID:   outboxID,
		TiketID:    rw.TiketID,
		LayananID:  rw.LayananID,
		UserID:     rw.UserID,
		LoketID:    rw.LoketID,
		StatusLama: rw.StatusLama,
		StatusBaru: rw.StatusBaru,
		Waktu:      rw.Waktu.UTC(),
	})
	if mongo.IsDuplicateKeyError(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("mencatat riwayat tiket %d: %w", rw.TiketID, err)
	}
	return true, nil
}

func (r *RiwayatRepository) ListByTiket(ctx context.Context, tiketID int64) ([]model.RiwayatTiket, error) {
	cur, err := r.col.Find(ctx,
		bson.D{{Key: "tiket_id", Value: tiketID}},
		options.Find().SetSort(bson.D{{Key: "waktu", Value: 1}, {Key: "outbox_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("mengambil riwayat tiket %d: %w", tiketID, err)
	}

	var docs []riwayatDocument
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("membaca riwayat tiket %d: %w", tiketID, err)
	}

	items := make([]model.RiwayatTiket, 0, len(docs))
	for _, d := range docs {
		items = append(items, d.toModel())
	}
	return items, nil
}

type statistikFacet struct {
	PerJam []struct {
		Jam    int `bson:"_id"`
		Jumlah int `bson:"jumlah"`
	} `bson:"per_jam"`
	PerStatus []struct {
		Status string `bson:"_id"`
		Jumlah int    `bson:"jumlah"`
	} `bson:"per_status"`
	Durasi []struct {
		RataTunggu  *float64 `bson:"rata_tunggu"`
		MaksTunggu  *float64 `bson:"maks_tunggu"`
		RataLayanan *float64 `bson:"rata_layanan"`
	} `bson:"durasi"`
}

func waktuStatus(status string) bson.D {
	return bson.D{{Key: "$cond", Value: bson.A{
		bson.D{{Key: "$eq", Value: bson.A{"$status_baru", status}}}, "$waktu", nil,
	}}}
}

func selisihJika(dari, sampai string) bson.D {
	return bson.D{{Key: "$cond", Value: bson.A{
		bson.D{{Key: "$and", Value: bson.A{dari, sampai}}},
		bson.D{{Key: "$subtract", Value: bson.A{sampai, dari}}},
		nil,
	}}}
}

func menit(ms *float64) float64 {
	if ms == nil {
		return 0
	}
	return *ms / 60000
}

func (r *RiwayatRepository) Statistik(ctx context.Context, layananID int64, dari, sampai time.Time, zona string) (model.StatistikLayanan, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "layanan_id", Value: layananID},
			{Key: "waktu", Value: bson.D{{Key: "$gte", Value: dari.UTC()}, {Key: "$lt", Value: sampai.UTC()}}},
		}}},
		{{Key: "$facet", Value: bson.D{
			{Key: "per_jam", Value: bson.A{
				bson.D{{Key: "$match", Value: bson.D{{Key: "status_baru", Value: "menunggu"}, {Key: "status_lama", Value: ""}}}},
				bson.D{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: bson.D{{Key: "$hour", Value: bson.D{{Key: "date", Value: "$waktu"}, {Key: "timezone", Value: zona}}}}},
					{Key: "jumlah", Value: bson.D{{Key: "$sum", Value: 1}}},
				}}},
				bson.D{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
			}},
			{Key: "per_status", Value: bson.A{
				bson.D{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: "$status_baru"},
					{Key: "jumlah", Value: bson.D{{Key: "$sum", Value: 1}}},
				}}},
			}},
			{Key: "durasi", Value: bson.A{
				bson.D{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: "$tiket_id"},
					{Key: "diambil", Value: bson.D{{Key: "$min", Value: waktuStatus("menunggu")}}},
					{Key: "dipanggil", Value: bson.D{{Key: "$min", Value: waktuStatus("dipanggil")}}},
					{Key: "selesai", Value: bson.D{{Key: "$max", Value: waktuStatus("selesai")}}},
				}}},
				bson.D{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: nil},
					{Key: "rata_tunggu", Value: bson.D{{Key: "$avg", Value: selisihJika("$diambil", "$dipanggil")}}},
					{Key: "maks_tunggu", Value: bson.D{{Key: "$max", Value: selisihJika("$diambil", "$dipanggil")}}},
					{Key: "rata_layanan", Value: bson.D{{Key: "$avg", Value: selisihJika("$dipanggil", "$selesai")}}},
				}}},
			}},
		}}},
	}

	cur, err := r.col.Aggregate(ctx, pipeline)
	if err != nil {
		return model.StatistikLayanan{}, fmt.Errorf("menghitung statistik layanan %d: %w", layananID, err)
	}

	var hasil []statistikFacet
	if err := cur.All(ctx, &hasil); err != nil {
		return model.StatistikLayanan{}, fmt.Errorf("membaca statistik layanan %d: %w", layananID, err)
	}

	stat := model.StatistikLayanan{
		LayananID:       layananID,
		JumlahPerStatus: map[string]int{},
		PerJam:          []model.StatistikJam{},
	}
	if len(hasil) == 0 {
		return stat, nil
	}

	f := hasil[0]
	terbanyak := 0
	for _, j := range f.PerJam {
		stat.PerJam = append(stat.PerJam, model.StatistikJam{Jam: j.Jam, Jumlah: j.Jumlah})
		stat.TotalTiket += j.Jumlah
		if j.Jumlah > terbanyak {
			terbanyak = j.Jumlah
			jam := j.Jam
			stat.JamTersibuk = &jam
		}
	}
	for _, s := range f.PerStatus {
		stat.JumlahPerStatus[s.Status] = s.Jumlah
	}
	if len(f.Durasi) > 0 {
		stat.RataTungguMenit = menit(f.Durasi[0].RataTunggu)
		stat.MaksTungguMenit = menit(f.Durasi[0].MaksTunggu)
		stat.RataLayananMenit = menit(f.Durasi[0].RataLayanan)
	}
	return stat, nil
}
