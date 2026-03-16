package feeder

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitdev.devops.krungthai.com/starwolf/backend/common/app"
	"github.com/11SF/dogjohn-be/app/feeder/access"
	mocks "github.com/11SF/dogjohn-be/app/feeder/access/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestHandler(t *testing.T) (*handler, *mocks.FeederRepositoryMock) {
	repoMock := mocks.NewFeederRepositoryMock(t)
	h := NewHandler(HandlerConfig{FeederRepo: repoMock})
	return h, repoMock
}

func TestGetFeederAvailability_Available_ShouldReturn200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock := newTestHandler(t)

	reason := "เครื่องพร้อม"
	repoMock.EXPECT().GetAvailability(mock.Anything).Return(
		&access.FeederAvailabilityResponse{Available: true, Reason: &reason}, nil,
	)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/feeder/availability", nil)

	h.GetFeederAvailability(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp app.Response[FeederAvailabilityResponse]
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Data.Available)
}

func TestGetFeederAvailability_Unavailable_ShouldReturn200WithReason(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock := newTestHandler(t)

	reason := "เครื่องขัดข้อง"
	repoMock.EXPECT().GetAvailability(mock.Anything).Return(
		&access.FeederAvailabilityResponse{Available: false, Reason: &reason}, nil,
	)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/feeder/availability", nil)

	h.GetFeederAvailability(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp app.Response[FeederAvailabilityResponse]
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp.Data.Available)
	assert.Equal(t, "เครื่องขัดข้อง", *resp.Data.Reason)
}

func TestGetFeederAvailability_DBError_ShouldReturn500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, repoMock := newTestHandler(t)

	repoMock.EXPECT().GetAvailability(mock.Anything).Return(nil, assert.AnError)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/feeder/availability", nil)

	h.GetFeederAvailability(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
