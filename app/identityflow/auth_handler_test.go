package identityflow

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitdev.devops.krungthai.com/starwolf/backend/common/app"
	"gitdev.devops.krungthai.com/starwolf/backend/common/httpclient"
	"github.com/11SF/dogjohn-be/app/identityflow/access"
	access_mocks "github.com/11SF/dogjohn-be/app/identityflow/access/mocks"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAuthenticate_ExistingUser_ShouldReturn200(t *testing.T) {
	gin.SetMode(gin.TestMode)

	memberID := uuid.New()
	organizationID := uuid.New()

	authMock := access_mocks.NewAuthClientMock(t)
	memberMock := access_mocks.NewMemberClientMock(t)
	orgMock := access_mocks.NewOrganizationClientMock(t)

	h := NewHandler(HandlerConfig{
		AuthClient:         authMock,
		MemberClient:       memberMock,
		OrganizationClient: orgMock,
	})

	googleToken := "valid-google-token"

	authMock.EXPECT().ResolveIdentity(
		mock.Anything,
		access.ResolveIdentityRequest{GoogleAccessToken: googleToken},
	).Return(httpclient.Response[app.Response[access.ResolveIdentityResponse]]{
		Code: http.StatusOK,
		Response: app.Response[access.ResolveIdentityResponse]{
			Code:    app.CodeSuccess,
			Message: app.MessageSuccess,
			Data: &access.ResolveIdentityResponse{
				IsMember:       true,
				MemberID:       memberID,
				Username:       "John Doe",
				Email:          "john@example.com",
				ProfileImage:   "https://example.com/avatar.jpg",
				OrganizationID: organizationID,
				Role:           access.OrganizationMemberRoleAdmin,
			},
		},
	}, nil)

	authMock.EXPECT().IssueToken(
		mock.Anything,
		access.IssueTokenRequest{
			MemberID:       memberID,
			OrganizationID: organizationID,
			MemberRole:     access.OrganizationMemberRoleAdmin,
		},
	).Return(httpclient.Response[app.Response[access.IssueTokenResponse]]{
		Code: http.StatusOK,
		Response: app.Response[access.IssueTokenResponse]{
			Code:    app.CodeSuccess,
			Message: app.MessageSuccess,
			Data: &access.IssueTokenResponse{
				AccessToken: "jwt-access-token",
			},
		},
	}, nil)

	body, _ := json.Marshal(AuthRequest{GoogleAccessToken: googleToken})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/identity-flow/auth", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Authenticate(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp app.Response[AuthResponse]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, string(app.CodeSuccess), string(resp.Code))
	assert.NotNil(t, resp.Data)
	assert.Equal(t, "John Doe", resp.Data.Username)
	assert.Equal(t, "john@example.com", resp.Data.Email)
	assert.Equal(t, string(access.OrganizationMemberRoleAdmin), resp.Data.Role)
	assert.Equal(t, "https://example.com/avatar.jpg", resp.Data.ProfileImage)
	assert.Equal(t, "jwt-access-token", resp.Data.AccessToken)
}

func TestAuthenticate_InvalidRequestBody_ShouldReturn400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authMock := access_mocks.NewAuthClientMock(t)
	memberMock := access_mocks.NewMemberClientMock(t)
	orgMock := access_mocks.NewOrganizationClientMock(t)

	h := NewHandler(HandlerConfig{
		AuthClient:         authMock,
		MemberClient:       memberMock,
		OrganizationClient: orgMock,
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/identity-flow/auth", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Authenticate(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp app.Response[AuthResponse]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, string(app.CodeBadRequest), string(resp.Code))
}

func TestAuthenticate_ResolveIdentityError_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authMock := access_mocks.NewAuthClientMock(t)
	memberMock := access_mocks.NewMemberClientMock(t)
	orgMock := access_mocks.NewOrganizationClientMock(t)

	h := NewHandler(HandlerConfig{
		AuthClient:         authMock,
		MemberClient:       memberMock,
		OrganizationClient: orgMock,
	})

	authMock.EXPECT().ResolveIdentity(mock.Anything, mock.Anything).
		Return(httpclient.Response[app.Response[access.ResolveIdentityResponse]]{}, assert.AnError)

	body, _ := json.Marshal(AuthRequest{GoogleAccessToken: "token"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/identity-flow/auth", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Authenticate(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp app.Response[AuthResponse]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, string(app.CodeInternalError), string(resp.Code))
}

func TestAuthenticate_ResolveIdentityUpstreamError_ShouldReturnUpstreamStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authMock := access_mocks.NewAuthClientMock(t)
	memberMock := access_mocks.NewMemberClientMock(t)
	orgMock := access_mocks.NewOrganizationClientMock(t)

	h := NewHandler(HandlerConfig{
		AuthClient:         authMock,
		MemberClient:       memberMock,
		OrganizationClient: orgMock,
	})

	authMock.EXPECT().ResolveIdentity(mock.Anything, mock.Anything).
		Return(httpclient.Response[app.Response[access.ResolveIdentityResponse]]{
			Code: http.StatusUnauthorized,
			Response: app.Response[access.ResolveIdentityResponse]{
				Code:    app.CodeUnauthorized,
				Message: app.MessageUnauthorized,
			},
		}, nil)

	body, _ := json.Marshal(AuthRequest{GoogleAccessToken: "invalid-token"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/identity-flow/auth", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Authenticate(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp app.Response[AuthResponse]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, string(app.CodeUnauthorized), string(resp.Code))
}

func TestAuthenticate_ResolveIdentityNilData_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authMock := access_mocks.NewAuthClientMock(t)
	memberMock := access_mocks.NewMemberClientMock(t)
	orgMock := access_mocks.NewOrganizationClientMock(t)

	h := NewHandler(HandlerConfig{
		AuthClient:         authMock,
		MemberClient:       memberMock,
		OrganizationClient: orgMock,
	})

	authMock.EXPECT().ResolveIdentity(mock.Anything, mock.Anything).
		Return(httpclient.Response[app.Response[access.ResolveIdentityResponse]]{
			Code: http.StatusOK,
			Response: app.Response[access.ResolveIdentityResponse]{
				Code:    app.CodeSuccess,
				Message: app.MessageSuccess,
				Data:    nil,
			},
		}, nil)

	body, _ := json.Marshal(AuthRequest{GoogleAccessToken: "token"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/identity-flow/auth", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Authenticate(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp app.Response[AuthResponse]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, string(app.CodeInternalError), string(resp.Code))
}

func TestAuthenticate_NewUser_ShouldAutoRegisterAndReturn200(t *testing.T) {
	gin.SetMode(gin.TestMode)

	memberID := uuid.New()
	organizationID := uuid.New()

	authMock := access_mocks.NewAuthClientMock(t)
	memberMock := access_mocks.NewMemberClientMock(t)
	orgMock := access_mocks.NewOrganizationClientMock(t)

	h := NewHandler(HandlerConfig{
		AuthClient:         authMock,
		MemberClient:       memberMock,
		OrganizationClient: orgMock,
	})

	googleToken := "new-user-google-token"

	authMock.EXPECT().ResolveIdentity(mock.Anything, access.ResolveIdentityRequest{GoogleAccessToken: googleToken}).
		Return(httpclient.Response[app.Response[access.ResolveIdentityResponse]]{
			Code: http.StatusOK,
			Response: app.Response[access.ResolveIdentityResponse]{
				Code:    app.CodeSuccess,
				Message: app.MessageSuccess,
				Data: &access.ResolveIdentityResponse{
					IsMember: false,
					Email:    "newuser@example.com",
					Username: "New User",
				},
			},
		}, nil)

	memberMock.EXPECT().RegisterMember(mock.Anything, access.RegisterMemberRequest{
		Email: "newuser@example.com",
		Name:  "New User",
	}).Return(httpclient.Response[app.Response[access.RegisterMemberResponse]]{
		Code: http.StatusOK,
		Response: app.Response[access.RegisterMemberResponse]{
			Code:    app.CodeSuccess,
			Message: app.MessageSuccess,
			Data: &access.RegisterMemberResponse{
				MemberID: memberID,
			},
		},
	}, nil)

	orgMock.EXPECT().RegisterOrganization(mock.Anything, access.RegisterOrganizationRequest{
		Email:    "newuser@example.com",
		Name:     "New User",
		MemberID: memberID,
	}).Return(httpclient.Response[app.Response[access.RegisterOrganizationResponse]]{
		Code: http.StatusOK,
		Response: app.Response[access.RegisterOrganizationResponse]{
			Code:    app.CodeSuccess,
			Message: app.MessageSuccess,
			Data: &access.RegisterOrganizationResponse{
				OrganizationID: organizationID,
				MemberID:       memberID,
				Role:           string(access.OrganizationMemberRoleAdmin),
			},
		},
	}, nil)

	authMock.EXPECT().IssueToken(mock.Anything, access.IssueTokenRequest{
		MemberID:       memberID,
		OrganizationID: organizationID,
		MemberRole:     access.OrganizationMemberRoleAdmin,
	}).Return(httpclient.Response[app.Response[access.IssueTokenResponse]]{
		Code: http.StatusOK,
		Response: app.Response[access.IssueTokenResponse]{
			Code:    app.CodeSuccess,
			Message: app.MessageSuccess,
			Data: &access.IssueTokenResponse{
				AccessToken: "new-user-jwt-token",
			},
		},
	}, nil)

	body, _ := json.Marshal(AuthRequest{GoogleAccessToken: googleToken})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/identity-flow/auth", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Authenticate(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp app.Response[AuthResponse]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, string(app.CodeSuccess), string(resp.Code))
	assert.NotNil(t, resp.Data)
	assert.Equal(t, "new-user-jwt-token", resp.Data.AccessToken)
}

func TestAuthenticate_RegisterMemberError_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authMock := access_mocks.NewAuthClientMock(t)
	memberMock := access_mocks.NewMemberClientMock(t)
	orgMock := access_mocks.NewOrganizationClientMock(t)

	h := NewHandler(HandlerConfig{
		AuthClient:         authMock,
		MemberClient:       memberMock,
		OrganizationClient: orgMock,
	})

	authMock.EXPECT().ResolveIdentity(mock.Anything, mock.Anything).
		Return(httpclient.Response[app.Response[access.ResolveIdentityResponse]]{
			Code: http.StatusOK,
			Response: app.Response[access.ResolveIdentityResponse]{
				Code:    app.CodeSuccess,
				Message: app.MessageSuccess,
				Data: &access.ResolveIdentityResponse{
					IsMember: false,
					Email:    "newuser@example.com",
					Username: "New User",
				},
			},
		}, nil)

	memberMock.EXPECT().RegisterMember(mock.Anything, mock.Anything).
		Return(httpclient.Response[app.Response[access.RegisterMemberResponse]]{}, assert.AnError)

	body, _ := json.Marshal(AuthRequest{GoogleAccessToken: "token"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/identity-flow/auth", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Authenticate(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp app.Response[AuthResponse]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, string(app.CodeInternalError), string(resp.Code))
}

func TestAuthenticate_RegisterOrganizationError_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	memberID := uuid.New()

	authMock := access_mocks.NewAuthClientMock(t)
	memberMock := access_mocks.NewMemberClientMock(t)
	orgMock := access_mocks.NewOrganizationClientMock(t)

	h := NewHandler(HandlerConfig{
		AuthClient:         authMock,
		MemberClient:       memberMock,
		OrganizationClient: orgMock,
	})

	authMock.EXPECT().ResolveIdentity(mock.Anything, mock.Anything).
		Return(httpclient.Response[app.Response[access.ResolveIdentityResponse]]{
			Code: http.StatusOK,
			Response: app.Response[access.ResolveIdentityResponse]{
				Code:    app.CodeSuccess,
				Message: app.MessageSuccess,
				Data: &access.ResolveIdentityResponse{
					IsMember: false,
					Email:    "newuser@example.com",
					Username: "New User",
				},
			},
		}, nil)

	memberMock.EXPECT().RegisterMember(mock.Anything, mock.Anything).
		Return(httpclient.Response[app.Response[access.RegisterMemberResponse]]{
			Code: http.StatusOK,
			Response: app.Response[access.RegisterMemberResponse]{
				Code:    app.CodeSuccess,
				Message: app.MessageSuccess,
				Data:    &access.RegisterMemberResponse{MemberID: memberID},
			},
		}, nil)

	orgMock.EXPECT().RegisterOrganization(mock.Anything, mock.Anything).
		Return(httpclient.Response[app.Response[access.RegisterOrganizationResponse]]{}, assert.AnError)

	body, _ := json.Marshal(AuthRequest{GoogleAccessToken: "token"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/identity-flow/auth", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Authenticate(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp app.Response[AuthResponse]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, string(app.CodeInternalError), string(resp.Code))
}

func TestAuthenticate_IssueTokenError_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	memberID := uuid.New()
	organizationID := uuid.New()

	authMock := access_mocks.NewAuthClientMock(t)
	memberMock := access_mocks.NewMemberClientMock(t)
	orgMock := access_mocks.NewOrganizationClientMock(t)

	h := NewHandler(HandlerConfig{
		AuthClient:         authMock,
		MemberClient:       memberMock,
		OrganizationClient: orgMock,
	})

	authMock.EXPECT().ResolveIdentity(mock.Anything, mock.Anything).
		Return(httpclient.Response[app.Response[access.ResolveIdentityResponse]]{
			Code: http.StatusOK,
			Response: app.Response[access.ResolveIdentityResponse]{
				Code:    app.CodeSuccess,
				Message: app.MessageSuccess,
				Data: &access.ResolveIdentityResponse{
					IsMember:       true,
					MemberID:       memberID,
					OrganizationID: organizationID,
					Role:           access.OrganizationMemberRoleAdmin,
				},
			},
		}, nil)

	authMock.EXPECT().IssueToken(mock.Anything, mock.Anything).
		Return(httpclient.Response[app.Response[access.IssueTokenResponse]]{}, assert.AnError)

	body, _ := json.Marshal(AuthRequest{GoogleAccessToken: "token"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/identity-flow/auth", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Authenticate(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp app.Response[AuthResponse]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, string(app.CodeInternalError), string(resp.Code))
}

func TestAuthenticate_IssueTokenUpstreamError_ShouldReturnUpstreamStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	memberID := uuid.New()
	organizationID := uuid.New()

	authMock := access_mocks.NewAuthClientMock(t)
	memberMock := access_mocks.NewMemberClientMock(t)
	orgMock := access_mocks.NewOrganizationClientMock(t)

	h := NewHandler(HandlerConfig{
		AuthClient:         authMock,
		MemberClient:       memberMock,
		OrganizationClient: orgMock,
	})

	authMock.EXPECT().ResolveIdentity(mock.Anything, mock.Anything).
		Return(httpclient.Response[app.Response[access.ResolveIdentityResponse]]{
			Code: http.StatusOK,
			Response: app.Response[access.ResolveIdentityResponse]{
				Code:    app.CodeSuccess,
				Message: app.MessageSuccess,
				Data: &access.ResolveIdentityResponse{
					IsMember:       true,
					MemberID:       memberID,
					OrganizationID: organizationID,
					Role:           access.OrganizationMemberRoleAdmin,
				},
			},
		}, nil)

	authMock.EXPECT().IssueToken(mock.Anything, mock.Anything).
		Return(httpclient.Response[app.Response[access.IssueTokenResponse]]{
			Code: http.StatusUnprocessableEntity,
			Response: app.Response[access.IssueTokenResponse]{
				Code:    app.CodeBadRequest,
				Message: app.MessageBadRequest,
			},
		}, nil)

	body, _ := json.Marshal(AuthRequest{GoogleAccessToken: "token"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/identity-flow/auth", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Authenticate(c)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestAuthenticate_IssueTokenNilData_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	memberID := uuid.New()
	organizationID := uuid.New()

	authMock := access_mocks.NewAuthClientMock(t)
	memberMock := access_mocks.NewMemberClientMock(t)
	orgMock := access_mocks.NewOrganizationClientMock(t)

	h := NewHandler(HandlerConfig{
		AuthClient:         authMock,
		MemberClient:       memberMock,
		OrganizationClient: orgMock,
	})

	authMock.EXPECT().ResolveIdentity(mock.Anything, mock.Anything).
		Return(httpclient.Response[app.Response[access.ResolveIdentityResponse]]{
			Code: http.StatusOK,
			Response: app.Response[access.ResolveIdentityResponse]{
				Code:    app.CodeSuccess,
				Message: app.MessageSuccess,
				Data: &access.ResolveIdentityResponse{
					IsMember:       true,
					MemberID:       memberID,
					OrganizationID: organizationID,
					Role:           access.OrganizationMemberRoleAdmin,
				},
			},
		}, nil)

	authMock.EXPECT().IssueToken(mock.Anything, mock.Anything).
		Return(httpclient.Response[app.Response[access.IssueTokenResponse]]{
			Code: http.StatusOK,
			Response: app.Response[access.IssueTokenResponse]{
				Code:    app.CodeSuccess,
				Message: app.MessageSuccess,
				Data:    nil,
			},
		}, nil)

	body, _ := json.Marshal(AuthRequest{GoogleAccessToken: "token"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/identity-flow/auth", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Authenticate(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp app.Response[AuthResponse]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, string(app.CodeInternalError), string(resp.Code))
}
