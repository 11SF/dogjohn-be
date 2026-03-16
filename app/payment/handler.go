package payment

import (
	"github.com/11SF/dogjohn-be/app/payment/access"
)

type HandlerConfig struct {
	PaymentRepo access.PaymentRepository
}

type handler struct {
	paymentRepo access.PaymentRepository
}

func NewHandler(cfg HandlerConfig) *handler {
	return &handler{
		paymentRepo: cfg.PaymentRepo,
	}
}
