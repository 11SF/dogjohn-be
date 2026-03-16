package access

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"gitdev.devops.krungthai.com/starwolf/backend/common/app"
	"gitdev.devops.krungthai.com/starwolf/backend/common/httpclient"
	"github.com/google/uuid"
)

type OrganizationClient interface {
	GetCurrent(ctx context.Context, req GetCurrentOrganizationRequest) (httpclient.Response[app.Response[GetCurrentOrganizationResponse]], error)
	RegisterOrganization(ctx context.Context, req RegisterOrganizationRequest) (httpclient.Response[app.Response[RegisterOrganizationResponse]], error)
}

type organizationClient struct {
	baseURL string
	client  *http.Client
}

func NewOrganizationClient(baseURL string, client *http.Client) OrganizationClient {
	return &organizationClient{
		baseURL: baseURL,
		client:  client,
	}
}

type OrganizationStatusType string

const (
	OrganizationStatusActive   OrganizationStatusType = "ACTIVE"
	OrganizationStatusInactive OrganizationStatusType = "INACTIVE"
)

type GetCurrentOrganizationRequest struct {
	MemberID uuid.UUID `json:"memberId"`
}

type GetCurrentOrganizationResponse struct {
	OrganizationID uuid.UUID                  `json:"organizationId"`
	Name           string                     `json:"name"`
	Status         OrganizationStatusType     `json:"status"`
	Role           OrganizationMemberRoleType `json:"role"`
	JoinedAt       *time.Time                 `json:"joinedAt,omitempty"`
	CreatedAt      time.Time                  `json:"createdAt"`
	UpdatedAt      time.Time                  `json:"updatedAt"`
}

func (c *organizationClient) GetCurrent(ctx context.Context, req GetCurrentOrganizationRequest) (httpclient.Response[app.Response[GetCurrentOrganizationResponse]], error) {
	url := c.baseURL + "/api/v1/platform/organization/current"
	return httpclient.Post[GetCurrentOrganizationRequest, app.Response[GetCurrentOrganizationResponse]](ctx, c.client, url, req)
}

// RegisterOrganizationRequest represents the request to register an organization
type RegisterOrganizationRequest struct {
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	MemberID uuid.UUID `json:"memberId"`
}

// RegisterOrganizationResponse represents the response from registering an organization
type RegisterOrganizationResponse struct {
	OrganizationID uuid.UUID `json:"organizationId"`
	MemberID       uuid.UUID `json:"memberId"`
	Role           string    `json:"role"`
}

func (c *organizationClient) RegisterOrganization(ctx context.Context, req RegisterOrganizationRequest) (httpclient.Response[app.Response[RegisterOrganizationResponse]], error) {
	url := fmt.Sprintf("%s/api/v1/platform/organization/register", c.baseURL)
	return httpclient.Post[RegisterOrganizationRequest, app.Response[RegisterOrganizationResponse]](ctx, c.client, url, req)
}
