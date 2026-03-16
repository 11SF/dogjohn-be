package identityflow

import (
	"net/http"

	"gitdev.devops.krungthai.com/starwolf/backend/common/app"
	"gitdev.devops.krungthai.com/starwolf/backend/common/wrapper"
	"github.com/11SF/dogjohn-be/app/identityflow/access"
	"github.com/gin-gonic/gin"
)

type AuthRequest struct {
	GoogleAccessToken string `json:"googleAccessToken" binding:"required"`
}

type AuthResponse struct {
	Username     string `json:"username"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	ProfileImage string `json:"profileImage"`
	AccessToken  string `json:"accessToken"`
}

func (h *handler) Authenticate(c *gin.Context) {
	ctx := c.Request.Context()

	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		wrapper.Respond(c, wrapper.ResponseOption[AuthResponse]{
			HTTPStatus: http.StatusBadRequest,
			Code:       app.CodeBadRequest,
			Message:    app.MessageBadRequest,
			Err:        err,
		})
		return
	}

	// [1] ResolveIdentity
	identityResp, err := h.authClient.ResolveIdentity(ctx, access.ResolveIdentityRequest{
		GoogleAccessToken: req.GoogleAccessToken,
	})
	if err != nil {
		wrapper.Respond(c, wrapper.ResponseOption[AuthResponse]{
			HTTPStatus: http.StatusInternalServerError,
			Code:       app.CodeInternalError,
			Message:    app.MessageInternalError,
			Err:        err,
		})
		return
	}

	if identityResp.Code >= 400 {
		wrapper.Respond(c, wrapper.ResponseOption[AuthResponse]{
			HTTPStatus: identityResp.Code,
			Code:       identityResp.Response.Code,
			Message:    identityResp.Response.Message,
		})
		return
	}

	if identityResp.Response.Data == nil {
		wrapper.Respond(c, wrapper.ResponseOption[AuthResponse]{
			HTTPStatus: http.StatusInternalServerError,
			Code:       app.CodeInternalError,
			Message:    app.MessageInternalError,
		})
		return
	}

	identity := identityResp.Response.Data
	memberID := identity.MemberID
	organizationID := identity.OrganizationID
	role := identity.Role

	// [2] Auto-register flow for new user
	if !identity.IsMember {
		// [2a] RegisterMember
		memberResp, err := h.memberClient.RegisterMember(ctx, access.RegisterMemberRequest{
			Email: identity.Email,
			Name:  identity.Username,
		})
		if err != nil {
			wrapper.Respond(c, wrapper.ResponseOption[AuthResponse]{
				HTTPStatus: http.StatusInternalServerError,
				Code:       app.CodeInternalError,
				Message:    app.MessageInternalError,
				Err:        err,
			})
			return
		}

		if memberResp.Response.Data == nil {
			wrapper.Respond(c, wrapper.ResponseOption[AuthResponse]{
				HTTPStatus: http.StatusInternalServerError,
				Code:       app.CodeInternalError,
				Message:    app.MessageInternalError,
			})
			return
		}

		memberID = memberResp.Response.Data.MemberID

		// [2b] RegisterOrganization
		orgResp, err := h.organizationClient.RegisterOrganization(ctx, access.RegisterOrganizationRequest{
			Email:    identity.Email,
			Name:     identity.Username,
			MemberID: memberID,
		})
		if err != nil {
			wrapper.Respond(c, wrapper.ResponseOption[AuthResponse]{
				HTTPStatus: http.StatusInternalServerError,
				Code:       app.CodeInternalError,
				Message:    app.MessageInternalError,
				Err:        err,
			})
			return
		}

		if orgResp.Response.Data == nil {
			wrapper.Respond(c, wrapper.ResponseOption[AuthResponse]{
				HTTPStatus: http.StatusInternalServerError,
				Code:       app.CodeInternalError,
				Message:    app.MessageInternalError,
			})
			return
		}

		organizationID = orgResp.Response.Data.OrganizationID
		role = access.OrganizationMemberRoleType(orgResp.Response.Data.Role)
	}

	// [3] IssueToken
	tokenResp, err := h.authClient.IssueToken(ctx, access.IssueTokenRequest{
		MemberID:       memberID,
		OrganizationID: organizationID,
		MemberRole:     role,
	})
	if err != nil {
		wrapper.Respond(c, wrapper.ResponseOption[AuthResponse]{
			HTTPStatus: http.StatusInternalServerError,
			Code:       app.CodeInternalError,
			Message:    app.MessageInternalError,
			Err:        err,
		})
		return
	}

	if tokenResp.Code >= 400 {
		wrapper.Respond(c, wrapper.ResponseOption[AuthResponse]{
			HTTPStatus: tokenResp.Code,
			Code:       tokenResp.Response.Code,
			Message:    tokenResp.Response.Message,
		})
		return
	}

	if tokenResp.Response.Data == nil {
		wrapper.Respond(c, wrapper.ResponseOption[AuthResponse]{
			HTTPStatus: http.StatusInternalServerError,
			Code:       app.CodeInternalError,
			Message:    app.MessageInternalError,
		})
		return
	}

	wrapper.Respond(c, wrapper.ResponseOption[AuthResponse]{
		HTTPStatus: http.StatusOK,
		Code:       app.CodeSuccess,
		Message:    app.MessageSuccess,
		Data: &AuthResponse{
			Username:     identity.Username,
			Email:        identity.Email,
			Role:         string(role),
			ProfileImage: identity.ProfileImage,
			AccessToken:  tokenResp.Response.Data.AccessToken,
		},
	})
}
