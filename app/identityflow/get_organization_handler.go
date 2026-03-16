package identityflow

import (
	"net/http"
	"time"

	"gitdev.devops.krungthai.com/starwolf/backend/common/app"
	"gitdev.devops.krungthai.com/starwolf/backend/common/appcontext"
	"gitdev.devops.krungthai.com/starwolf/backend/common/wrapper"
	"github.com/11SF/dogjohn-be/app/identityflow/access"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type (
	GetOrganizationResponse struct {
		OrganizationID uuid.UUID                         `json:"organizationId"`
		Name           string                            `json:"name"`
		Status         access.OrganizationStatusType     `json:"status"`
		Role           access.OrganizationMemberRoleType `json:"role"`
		JoinedAt       *time.Time                        `json:"joinedAt,omitempty"`
		CreatedAt      time.Time                         `json:"createdAt"`
		UpdatedAt      time.Time                         `json:"updatedAt"`
	}
)

func (h *handler) GetOrganization(c *gin.Context) {
	ctx := c.Request.Context()

	claims, err := appcontext.GetAuthClaims(c)
	if err != nil {
		wrapper.Respond(c, wrapper.ResponseOption[GetOrganizationResponse]{
			HTTPStatus: http.StatusBadRequest,
			Code:       app.CodeBadRequest,
			Message:    app.MessageBadRequest,
			Err:        err,
		})
		return
	}

	resp, err := h.organizationClient.GetCurrent(ctx, access.GetCurrentOrganizationRequest{
		MemberID: uuid.MustParse(claims.Sub),
	})
	if err != nil {
		wrapper.Respond(c, wrapper.ResponseOption[GetOrganizationResponse]{
			HTTPStatus: http.StatusInternalServerError,
			Code:       app.CodeInternalError,
			Message:    app.MessageInternalError,
			Err:        err,
		})
		return
	}

	if resp.Code >= 400 {
		wrapper.Respond(c, wrapper.ResponseOption[GetOrganizationResponse]{
			HTTPStatus: resp.Code,
			Code:       resp.Response.Code,
			Message:    resp.Response.Message,
		})
		return
	}

	if resp.Response.Data == nil {
		wrapper.Respond(c, wrapper.ResponseOption[GetOrganizationResponse]{
			HTTPStatus: http.StatusInternalServerError,
			Code:       app.CodeInternalError,
			Message:    app.MessageInternalError,
		})
		return
	}

	data := resp.Response.Data

	wrapper.Respond(c, wrapper.ResponseOption[GetOrganizationResponse]{
		HTTPStatus: http.StatusOK,
		Code:       app.CodeSuccess,
		Message:    app.MessageSuccess,
		Data: &GetOrganizationResponse{
			OrganizationID: data.OrganizationID,
			Name:           data.Name,
			Status:         data.Status,
			Role:           data.Role,
			JoinedAt:       data.JoinedAt,
			CreatedAt:      data.CreatedAt,
			UpdatedAt:      data.UpdatedAt,
		},
	})
}
