package feeder

import (
	"net/http"

	"github.com/11SF/go-common/response"
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
		response.NewGinResponseError(c, http.StatusInternalServerError,
			response.NewError(response.GenericError, "Internal Server Error"))
		return
	}

	response.NewGinResponse(c, http.StatusOK, FeederAvailabilityResponse{
		Available: data.Available,
		Reason:    data.Reason,
	})
}
