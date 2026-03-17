package access

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentRepository interface {
	GetDetails(ctx context.Context) (*PaymentDetailsResponse, error)
	GetPrice(ctx context.Context, priceID string) (*Price, error)
}

type paymentRepository struct {
	db *pgxpool.Pool
}

func NewPaymentRepository(db *pgxpool.Pool) PaymentRepository {
	return &paymentRepository{db: db}
}

type Price struct {
	ID          string  `json:"id"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

type PaymentDetailsResponse struct {
	PromptPayID string  `json:"promptPayId"`
	Prices      []Price `json:"prices"`
}

func (r *paymentRepository) GetDetails(ctx context.Context) (*PaymentDetailsResponse, error) {
	var promptPayID string
	row := r.db.QueryRow(ctx, `SELECT prompt_pay_id FROM payment_config LIMIT 1`)
	if err := row.Scan(&promptPayID); err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, price, description FROM price_options ORDER BY display_order ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []Price
	for rows.Next() {
		var p Price
		if err := rows.Scan(&p.ID, &p.Price, &p.Description); err != nil {
			return nil, err
		}
		prices = append(prices, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &PaymentDetailsResponse{
		PromptPayID: promptPayID,
		Prices:      prices,
	}, nil
}

func (r *paymentRepository) GetPrice(ctx context.Context, priceID string) (*Price, error) {
	var p Price
	row := r.db.QueryRow(ctx,
		`SELECT id, price, description FROM price_options WHERE id = $1`,
		priceID,
	)
	if err := row.Scan(&p.ID, &p.Price, &p.Description); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}
