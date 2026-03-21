package router

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/11SF/dogjohn-be/app/feeder"
	feederAccess "github.com/11SF/dogjohn-be/app/feeder/access"
	"github.com/11SF/dogjohn-be/app/order"
	orderAccess "github.com/11SF/dogjohn-be/app/order/access"
	"github.com/11SF/dogjohn-be/app/payment"
	paymentAccess "github.com/11SF/dogjohn-be/app/payment/access"
	"github.com/11SF/dogjohn-be/config"
	"github.com/11SF/go-common/logger"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gin-gonic/gin"
)

type routeDeps struct {
	cfg        config.Config
	httpClient *http.Client
	db         *pgxpool.Pool
}

// New constructs a gin.Engine with routes and middleware configured.
func New(cfg config.Config, version, commit string, timeoutDuration time.Duration) (*gin.Engine, func()) {
	r := gin.New()
	r.Use(gin.Recovery())

	if config.IsLocalEnv() {
		r.Use(gin.Logger())
	}

	r.GET("/liveness", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok", "version": version, "commit": commit}) })
	r.GET("/readiness", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	r.Use(logger.GinMiddleware())

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.DBName)
	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		panic(err)
	}

	deps := routeDeps{
		cfg:        cfg,
		httpClient: &http.Client{},
		db:         db,
	}

	registerOrderRoutes(r, deps)
	registerFeederRoutes(r, deps)
	registerPaymentRoutes(r, deps)

	return r, func() {
		db.Close()
	}
}

func registerOrderRoutes(r *gin.Engine, deps routeDeps) {
	orderRepo := orderAccess.NewOrderRepository(deps.db)
	paymentRepo := paymentAccess.NewPaymentRepository(deps.db)
	slipOKClient := orderAccess.NewSlipOKClient(
		deps.cfg.SlipOK.BaseURL,
		deps.cfg.SlipOK.BranchID,
		deps.cfg.SlipOK.APIKey,
		deps.httpClient,
	)
	haClient := orderAccess.NewHomeAssistantClient(
		deps.cfg.HomeAssistant.BaseURL,
		deps.cfg.HomeAssistant.Token,
		deps.cfg.HomeAssistant.EntityID,
		deps.httpClient,
	)

	h := order.NewHandler(order.HandlerConfig{
		Config:      deps.cfg,
		OrderRepo:   orderRepo,
		PaymentRepo: paymentRepo,
		SlipOK:      slipOKClient,
		HAClient:    haClient,
	})

	g := r.Group("/api/v1/order")
	{
		g.POST("/submit", h.SubmitOrder)
		g.GET("/summary", h.GetOrderSummary)
		g.GET("/history", h.GetOrderHistory)
		g.GET("/:orderId", h.GetOrder)
	}
}

func registerFeederRoutes(r *gin.Engine, deps routeDeps) {
	feederRepo := feederAccess.NewFeederRepository(deps.db)

	h := feeder.NewHandler(feeder.HandlerConfig{
		FeederRepo: feederRepo,
	})

	g := r.Group("/api/v1/feeder")
	{
		g.GET("/availability", h.GetFeederAvailability)
	}
}

func registerPaymentRoutes(r *gin.Engine, deps routeDeps) {
	paymentRepo := paymentAccess.NewPaymentRepository(deps.db)

	h := payment.NewHandler(payment.HandlerConfig{
		PaymentRepo: paymentRepo,
	})

	g := r.Group("/api/v1/payment")
	{
		g.GET("/details", h.GetPaymentDetails)
	}
}

func allowedHeaders(refIDHeaderKey string) []string {
	return []string{
		"Content-Type",
		"Content-Length",
		"Accept-Encoding",
		"X-CSRF-Token",
		"Authorization",
		"accept",
		"origin",
		"Cache-Control",
		"X-Requested-With",
		refIDHeaderKey,
	}
}
