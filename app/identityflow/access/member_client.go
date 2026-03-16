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

type MemberClient interface {
	GetMe(ctx context.Context, req GetMemberMeRequest) (httpclient.Response[app.Response[GetMemberMeResponse]], error)
	RegisterMember(ctx context.Context, req RegisterMemberRequest) (httpclient.Response[app.Response[RegisterMemberResponse]], error)
}

type memberClient struct {
	baseURL string
	client  *http.Client
}

func NewMemberClient(baseURL string, client *http.Client) MemberClient {
	return &memberClient{
		baseURL: baseURL,
		client:  client,
	}
}

type GetMemberMeRequest struct {
	MemberID uuid.UUID `json:"memberId"`
}

type GetMemberMeResponse struct {
	MemberID    uuid.UUID        `json:"memberId"`
	Username    string           `json:"username"`
	Email       string           `json:"email"`
	HashedEmail string           `json:"hashedEmail"`
	Status      MemberStatusType `json:"status"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

func (c *memberClient) GetMe(ctx context.Context, req GetMemberMeRequest) (httpclient.Response[app.Response[GetMemberMeResponse]], error) {
	url := c.baseURL + "/api/v1/platform/member/me"
	return httpclient.Post[GetMemberMeRequest, app.Response[GetMemberMeResponse]](ctx, c.client, url, req)
}

// RegisterMemberRequest represents the request to register a member
type RegisterMemberRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// RegisterMemberResponse represents the response from registering a member
type RegisterMemberResponse struct {
	MemberID uuid.UUID `json:"memberId"`
}

func (c *memberClient) RegisterMember(ctx context.Context, req RegisterMemberRequest) (httpclient.Response[app.Response[RegisterMemberResponse]], error) {
	url := fmt.Sprintf("%s/api/v1/platform/member/register", c.baseURL)
	return httpclient.Post[RegisterMemberRequest, app.Response[RegisterMemberResponse]](ctx, c.client, url, req)
}
