package access

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FeederRepository interface {
	GetAvailability(ctx context.Context) (*FeederAvailabilityResponse, error)
}

type feederRepository struct {
	db *pgxpool.Pool
}

func NewFeederRepository(db *pgxpool.Pool) FeederRepository {
	return &feederRepository{db: db}
}

type FeederAvailabilityResponse struct {
	Available bool    `json:"available"`
	Reason    *string `json:"reason"`
}

func (r *feederRepository) GetAvailability(ctx context.Context) (*FeederAvailabilityResponse, error) {
	var available bool
	var reason *string

	row := r.db.QueryRow(ctx,
		`SELECT available, reason FROM feeder_status ORDER BY updated_at DESC LIMIT 1`,
	)
	if err := row.Scan(&available, &reason); err != nil {
		return nil, err
	}

	return &FeederAvailabilityResponse{
		Available: available,
		Reason:    reason,
	}, nil
}
