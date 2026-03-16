package order

import (
	"github.com/11SF/dogjohn-be/app/order/access"
)

type HandlerConfig struct {
	OrderRepo   access.OrderRepository
	SlipOK      access.SlipOKClient
	HAClient    access.HomeAssistantClient
}

type handler struct {
	orderRepo access.OrderRepository
	slipOK    access.SlipOKClient
	haClient  access.HomeAssistantClient
}

func NewHandler(cfg HandlerConfig) *handler {
	return &handler{
		orderRepo: cfg.OrderRepo,
		slipOK:    cfg.SlipOK,
		haClient:  cfg.HAClient,
	}
}
