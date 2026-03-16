package feeder

import (
	"github.com/11SF/dogjohn-be/app/feeder/access"
)

type HandlerConfig struct {
	FeederRepo access.FeederRepository
}

type handler struct {
	feederRepo access.FeederRepository
}

func NewHandler(cfg HandlerConfig) *handler {
	return &handler{
		feederRepo: cfg.FeederRepo,
	}
}
