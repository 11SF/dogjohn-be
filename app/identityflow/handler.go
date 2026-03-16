package identityflow

import (
	"github.com/11SF/dogjohn-be/app/identityflow/access"
)

type HandlerConfig struct {
	AuthClient         access.AuthClient
	MemberClient       access.MemberClient
	OrganizationClient access.OrganizationClient
}

type handler struct {
	authClient         access.AuthClient
	memberClient       access.MemberClient
	organizationClient access.OrganizationClient
}

func NewHandler(cfg HandlerConfig) *handler {
	return &handler{
		authClient:         cfg.AuthClient,
		memberClient:       cfg.MemberClient,
		organizationClient: cfg.OrganizationClient,
	}
}
