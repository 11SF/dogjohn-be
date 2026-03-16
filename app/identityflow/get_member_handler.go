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
	GetMemberResponse struct {
		MemberID    uuid.UUID               `json:"memberId"`
		Username    string                  `json:"username"`
		Email       string                  `json:"email"`
		HashedEmail string                  `json:"hashedEmail"`
		Status      access.MemberStatusType `json:"status"`
		CreatedAt   time.Time               `json:"createdAt"`
		UpdatedAt   time.Time               `json:"updatedAt"`
	}
)

func (h *handler) GetMember(c *gin.Context) {
	ctx := c.Request.Context()

	claims, err := appcontext.GetAuthClaims(c)
	if err != nil {
		wrapper.Respond(c, wrapper.ResponseOption[GetMemberResponse]{
			HTTPStatus: http.StatusBadRequest,
			Code:       app.CodeBadRequest,
			Message:    app.MessageBadRequest,
			Err:        err,
		})
		return
	}

	resp, err := h.memberClient.GetMe(ctx, access.GetMemberMeRequest{
		MemberID: uuid.MustParse(claims.Sub),
	})
	if err != nil {
		wrapper.Respond(c, wrapper.ResponseOption[GetMemberResponse]{
			HTTPStatus: http.StatusInternalServerError,
			Code:       app.CodeInternalError,
			Message:    app.MessageInternalError,
			Err:        err,
		})
		return
	}

	if resp.Code >= 400 {
		wrapper.Respond(c, wrapper.ResponseOption[GetMemberResponse]{
			HTTPStatus: resp.Code,
			Code:       resp.Response.Code,
			Message:    resp.Response.Message,
		})
		return
	}

	if resp.Response.Data == nil {
		wrapper.Respond(c, wrapper.ResponseOption[GetMemberResponse]{
			HTTPStatus: http.StatusInternalServerError,
			Code:       app.CodeInternalError,
			Message:    app.MessageInternalError,
		})
		return
	}

	data := resp.Response.Data

	wrapper.Respond(c, wrapper.ResponseOption[GetMemberResponse]{
		HTTPStatus: http.StatusOK,
		Code:       app.CodeSuccess,
		Message:    app.MessageSuccess,
		Data: &GetMemberResponse{
			MemberID:    data.MemberID,
			Username:    data.Username,
			Email:       data.Email,
			HashedEmail: data.HashedEmail,
			Status:      data.Status,
			CreatedAt:   data.CreatedAt,
			UpdatedAt:   data.UpdatedAt,
		},
	})
}
