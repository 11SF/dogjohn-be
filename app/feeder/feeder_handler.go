package feeder

import (
	"net/http"

	"gitdev.devops.krungthai.com/starwolf/backend/common/app"
	"gitdev.devops.krungthai.com/starwolf/backend/common/wrapper"
	"github.com/gin-gonic/gin"
)

type FeederAvailabilityResponse struct {
	Available bool    `json:"available"`
	Reason    *string `json:"reason"`
}

func (h *handler) GetFeederAvailability(c *gin.Context) {
	ctx := c.Request.Context()

	data, err := h.feederRepo.GetAvailability(ctx)
	if err != nil {
		wrapper.Respond(c, wrapper.ResponseOption[FeederAvailabilityResponse]{
			HTTPStatus: http.StatusInternalServerError,
			Code:       app.CodeInternalError,
			Message:    app.MessageInternalError,
			Err:        err,
		})
		return
	}

	wrapper.Respond(c, wrapper.ResponseOption[FeederAvailabilityResponse]{
		HTTPStatus: http.StatusOK,
		Code:       app.CodeSuccess,
		Message:    app.MessageSuccess,
		Data: &FeederAvailabilityResponse{
			Available: data.Available,
			Reason:    data.Reason,
		},
	})
}
