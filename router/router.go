package router

import (
	"net/http"
	"time"

	"gitdev.devops.krungthai.com/starwolf/backend/common/app"
	"gitdev.devops.krungthai.com/starwolf/backend/common/database"
	"gitdev.devops.krungthai.com/starwolf/backend/common/health"
	"gitdev.devops.krungthai.com/starwolf/backend/common/httpclient"
	"gitdev.devops.krungthai.com/starwolf/backend/common/middleware"
	"gitdev.devops.krungthai.com/starwolf/backend/common/token"

	"github.com/11SF/dogjohn-be/app/feeder"
	feederAccess "github.com/11SF/dogjohn-be/app/feeder/access"
	"github.com/11SF/dogjohn-be/app/order"
	orderAccess "github.com/11SF/dogjohn-be/app/order/access"
	"github.com/11SF/dogjohn-be/app/payment"
	paymentAccess "github.com/11SF/dogjohn-be/app/payment/access"
	"github.com/11SF/dogjohn-be/config"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gin-gonic/gin"
)

type routeDeps struct {
	cfg         config.Config
	httpClient  *http.Client
	jwtVerifier token.JWTVerifier
	jwtParser   token.JWTParser
	db          *pgxpool.Pool
}

// New constructs a gin.Engine with routes and middleware configured.
func New(cfg config.Config, version, commit string, timeoutDuration time.Duration) (*gin.Engine, func()) {
	r := gin.New()
	r.Use(gin.Recovery())

	if config.IsLocalEnv() {
		r.Use(gin.Logger())
	}

	r.GET("/liveness", health.Liveness(version, commit))
	r.GET("/metrics", health.Metrics())
	r.GET("/readiness", health.Readiness())

	r.Use(
		middleware.SecurityHeaders(),
		middleware.AccessControl(cfg.AccessControl.AllowOrigin, allowedHeaders(cfg.Header.RefIDHeaderKey)),
		app.TraceContextTraceIDMiddleware(""),
		app.RefIDMiddleware(cfg.Header.RefIDHeaderKey),
		app.AutoLoggingMiddleware,
		middleware.Timeout(timeoutDuration),
		middleware.AccessLog(),
	)

	httpClient := httpclient.NewHTTPClient(app.ForwardRefIDOption)

	db := database.NewPostgresDB(database.PostgresConfig{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
	})

	deps := routeDeps{
		cfg:        cfg,
		httpClient: httpClient,
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
		OrderRepo: orderRepo,
		SlipOK:    slipOKClient,
		HAClient:  haClient,
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

// func newJWTVerifier(cfg config.Config) token.JWTVerifier {
// 	return token.MustNewJWTVerifier(token.JWTVerifierConfig{
// 		PublicKey: cfg.JWT.PublicKey,
// 		Alg:       string(token.ES256),
// 	})
// }

// func newJWTParser(cfg config.Config) token.JWTParser {
// 	return token.MustNewJWTParser(token.JWTParserConfig{
// 		Issuer:   cfg.JWT.Issuer,
// 		Audience: cfg.JWT.Audience,
// 	})
// }

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
