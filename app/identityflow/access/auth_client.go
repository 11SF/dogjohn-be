package access

import (
	"context"
	"net/http"

	"gitdev.devops.krungthai.com/starwolf/backend/common/app"
	"gitdev.devops.krungthai.com/starwolf/backend/common/httpclient"
	"github.com/google/uuid"
)

type AuthClient interface {
	ResolveIdentity(ctx context.Context, req ResolveIdentityRequest) (httpclient.Response[app.Response[ResolveIdentityResponse]], error)
	IssueToken(ctx context.Context, req IssueTokenRequest) (httpclient.Response[app.Response[IssueTokenResponse]], error)
}

type authClient struct {
	baseURL string
	client  *http.Client
}

func NewAuthClient(baseURL string, client *http.Client) AuthClient {
	return &authClient{
		baseURL: baseURL,
		client:  client,
	}
}

type ResolveIdentityRequest struct {
	GoogleAccessToken string `json:"googleAccessToken"`
}

type ResolveIdentityResponse struct {
	IsMember       bool                       `json:"isMember"`
	MemberID       uuid.UUID                  `json:"memberId"`
	Username       string                     `json:"username"`
	Email          string                     `json:"email"`
	HashedEmail    string                     `json:"hashedEmail"`
	Status         MemberStatusType           `json:"status"`
	OrganizationID uuid.UUID                  `json:"organizationID"`
	Role           OrganizationMemberRoleType `json:"role"`
	ProfileImage   string                     `json:"profileImage"`
}

type MemberStatusType string

const (
	MemberStatusActive    MemberStatusType = "ACTIVE"
	MemberStatusSuspended MemberStatusType = "SUSPENDED"
)

type OrganizationMemberRoleType string

const (
	OrganizationMemberRoleAdmin OrganizationMemberRoleType = "ADMIN"
	OrganizationMemberRoleUser  OrganizationMemberRoleType = "USER"
)

type IssueTokenRequest struct {
	MemberID       uuid.UUID                  `json:"memberId"`
	OrganizationID uuid.UUID                  `json:"organizationId"`
	MemberRole     OrganizationMemberRoleType `json:"memberRole"`
}

type IssueTokenResponse struct {
	AccessToken string `json:"accessToken"`
}

func (c *authClient) ResolveIdentity(ctx context.Context, req ResolveIdentityRequest) (httpclient.Response[app.Response[ResolveIdentityResponse]], error) {
	url := c.baseURL + "/api/v1/platform/auth/resolve-identity"
	return httpclient.Post[ResolveIdentityRequest, app.Response[ResolveIdentityResponse]](ctx, c.client, url, req)
}

func (c *authClient) IssueToken(ctx context.Context, req IssueTokenRequest) (httpclient.Response[app.Response[IssueTokenResponse]], error) {
	url := c.baseURL + "/api/v1/platform/auth/issue-token"
	return httpclient.Post[IssueTokenRequest, app.Response[IssueTokenResponse]](ctx, c.client, url, req)
}
