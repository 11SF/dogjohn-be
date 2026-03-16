package access

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type OrderRepository interface {
	SubmitOrder(ctx context.Context, req SubmitOrderRequest) (*SubmitOrderResponse, error)
	GetOrder(ctx context.Context, orderID string) (*GetOrderResponse, error)
	GetOrderSummary(ctx context.Context) (*OrderSummaryResponse, error)
	GetOrderHistory(ctx context.Context, req OrderHistoryRequest) (*OrderHistoryResponse, error)
	SavePaymentTxnLog(ctx context.Context, orderID string, response any) error
	UpdateOrderProcessing(ctx context.Context, orderID, paymentTxnRef string) error
	UpdateOrderFailed(ctx context.Context, orderID, reason string) error
	IsPaymentTxnRefDuplicate(ctx context.Context, txnRef string) (bool, error)
}

type orderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) OrderRepository {
	return &orderRepository{db: db}
}

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "PENDING"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusCompleted  OrderStatus = "COMPLETED"
	OrderStatusFailed     OrderStatus = "FAILED"
)

type PriceDetails struct {
	Price       string `json:"price"`
	Description string `json:"description"`
}

type SubmitOrderRequest struct {
	PriceID      string
	CustomerName string
	SlipImage    []byte
}

type SubmitOrderResponse struct {
	OrderID string `json:"orderId"`
}

type GetOrderResponse struct {
	OrderID        string       `json:"orderId"`
	CustomerName   string       `json:"customerName"`
	PriceDetails   PriceDetails `json:"priceDetails"`
	OrderStatus    OrderStatus  `json:"orderStatus"`
	FailureReason  *string      `json:"failureReason"`
	PaymentTxnRef  string       `json:"paymentTxnRef"`
	OrderedAt      int64        `json:"orderedAt"`
}

type OrderSummaryResponse struct {
	Today       int     `json:"today"`
	DogCount    int     `json:"dogCount"`
	TotalAmount float64 `json:"totalAmount"`
}

type OrderHistoryRequest struct {
	Limit  int
	Offset int
}

type OrderHistoryItem struct {
	OrderID      string       `json:"orderId"`
	CustomerName string       `json:"customerName"`
	PriceDetails PriceDetails `json:"priceDetails"`
	OrderStatus  OrderStatus  `json:"orderStatus"`
	OrderedAt    int64        `json:"orderedAt"`
}

type OrderHistoryResponse struct {
	Items []OrderHistoryItem `json:"items"`
	Total int                `json:"total"`
}

func (r *orderRepository) SubmitOrder(ctx context.Context, req SubmitOrderRequest) (*SubmitOrderResponse, error) {
	var orderID string
	err := r.db.QueryRow(ctx,
		`INSERT INTO orders (customer_name, price_id, slip_image, order_status, ordered_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		req.CustomerName,
		req.PriceID,
		req.SlipImage,
		string(OrderStatusPending),
		time.Now().Unix(),
	).Scan(&orderID)
	if err != nil {
		return nil, err
	}
	return &SubmitOrderResponse{OrderID: orderID}, nil
}

func (r *orderRepository) GetOrder(ctx context.Context, orderID string) (*GetOrderResponse, error) {
	var resp GetOrderResponse
	var status string

	row := r.db.QueryRow(ctx,
		`SELECT o.id, o.customer_name, p.price::text, p.description,
		        o.order_status, o.failure_reason, o.payment_txn_ref, o.ordered_at
		 FROM orders o
		 JOIN price_options p ON o.price_id = p.id
		 WHERE o.id = $1`,
		orderID,
	)
	err := row.Scan(
		&resp.OrderID,
		&resp.CustomerName,
		&resp.PriceDetails.Price,
		&resp.PriceDetails.Description,
		&status,
		&resp.FailureReason,
		&resp.PaymentTxnRef,
		&resp.OrderedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	resp.OrderStatus = OrderStatus(status)
	return &resp, nil
}

func (r *orderRepository) GetOrderSummary(ctx context.Context) (*OrderSummaryResponse, error) {
	var resp OrderSummaryResponse
	err := r.db.QueryRow(ctx,
		`SELECT
		    COUNT(*) FILTER (WHERE to_timestamp(ordered_at)::date = CURRENT_DATE) AS today,
		    COALESCE((SELECT dog_count FROM feeder_config LIMIT 1), 0)            AS dog_count,
		    COALESCE(SUM(p.price) FILTER (WHERE o.order_status = 'COMPLETED'), 0) AS total_amount
		 FROM orders o
		 JOIN price_options p ON o.price_id = p.id`,
	).Scan(&resp.Today, &resp.DogCount, &resp.TotalAmount)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (r *orderRepository) GetOrderHistory(ctx context.Context, req OrderHistoryRequest) (*OrderHistoryResponse, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var total int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM orders`).Scan(&total); err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx,
		`SELECT o.id, o.customer_name, p.price::text, p.description, o.order_status, o.ordered_at
		 FROM orders o
		 JOIN price_options p ON o.price_id = p.id
		 ORDER BY o.ordered_at DESC
		 LIMIT $1 OFFSET $2`,
		req.Limit,
		req.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []OrderHistoryItem
	for rows.Next() {
		var item OrderHistoryItem
		var status string
		if err := rows.Scan(
			&item.OrderID, &item.CustomerName,
			&item.PriceDetails.Price, &item.PriceDetails.Description,
			&status, &item.OrderedAt,
		); err != nil {
			return nil, err
		}
		item.OrderStatus = OrderStatus(status)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if items == nil {
		items = []OrderHistoryItem{}
	}

	return &OrderHistoryResponse{Items: items, Total: total}, nil
}

func (r *orderRepository) UpdateOrderProcessing(ctx context.Context, orderID, paymentTxnRef string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE orders SET order_status = $2, payment_txn_ref = $3 WHERE id = $1`,
		orderID, string(OrderStatusProcessing), paymentTxnRef,
	)
	return err
}

func (r *orderRepository) UpdateOrderFailed(ctx context.Context, orderID, reason string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE orders SET order_status = $2, failure_reason = $3 WHERE id = $1`,
		orderID, string(OrderStatusFailed), reason,
	)
	return err
}

func (r *orderRepository) IsPaymentTxnRefDuplicate(ctx context.Context, txnRef string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(
		    SELECT 1 FROM orders
		    WHERE payment_txn_ref = $1 AND order_status != $2
		)`,
		txnRef, string(OrderStatusFailed),
	).Scan(&exists)
	return exists, err
}

func (r *orderRepository) SavePaymentTxnLog(ctx context.Context, orderID string, response any) error {
	b, err := json.Marshal(response)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`INSERT INTO payment_txn_logs (order_id, response) VALUES ($1, $2)`,
		orderID, b,
	)
	return err
}
