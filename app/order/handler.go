package order

import (
	"github.com/11SF/dogjohn-be/app/order/access"
	paymentaccess "github.com/11SF/dogjohn-be/app/payment/access"
	"github.com/11SF/dogjohn-be/config"
)

type HandlerConfig struct {
	Config      config.Config
	OrderRepo   access.OrderRepository
	PaymentRepo paymentaccess.PaymentRepository
	SlipOK      access.SlipOKClient
	HAClient    access.HomeAssistantClient
}

type handler struct {
	config      config.Config
	orderRepo   access.OrderRepository
	paymentRepo paymentaccess.PaymentRepository
	slipOK      access.SlipOKClient
	haClient    access.HomeAssistantClient
}

func NewHandler(cfg HandlerConfig) *handler {
	return &handler{
		config:      cfg.Config,
		orderRepo:   cfg.OrderRepo,
		paymentRepo: cfg.PaymentRepo,
		slipOK:      cfg.SlipOK,
		haClient:    cfg.HAClient,
	}
}
