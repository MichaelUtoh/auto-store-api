package router

import (
	"auto-store-api/internal/config"
	"auto-store-api/internal/handlers"
	"auto-store-api/internal/middleware"
	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"
	"auto-store-api/internal/services"
	"auto-store-api/internal/validators"
	"auto-store-api/pkg/auth"
	"auto-store-api/pkg/cache"
	"auto-store-api/pkg/email"
	"auto-store-api/pkg/storage"
	"net/http"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"gorm.io/gorm"

	_ "auto-store-api/docs"
)

// Setup builds and returns the Gin engine with all routes and middleware
func Setup(cfg *config.Config, db *gorm.DB, log *zap.Logger) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	validators.RegisterGin()
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Recovery(log))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.CORS.Origins))
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	if cfg.Upload.MaxSize > 0 {
		r.Use(middleware.RequestSizeLimit(cfg.Upload.MaxSize))
	}

	generalLimiter := middleware.NewRateLimiter(cfg.RateLimit.RPM, cfg.RateLimit.RPM)
	authLimiter := middleware.NewRateLimiter(cfg.RateLimit.AuthRPM, cfg.RateLimit.AuthRPM)

	jwt := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry)

	userRepo := repositories.NewUserRepository(db)
	productRepo := repositories.NewProductRepository(db)
	categoryRepo := repositories.NewCategoryRepository(db)
	compatRepo := repositories.NewVehicleCompatibilityRepository(db)
	orderRepo := repositories.NewOrderRepository(db)
	cartRepo := repositories.NewCartRepository(db)
	wishlistRepo := repositories.NewWishlistRepository(db)
	addressRepo := repositories.NewAddressRepository(db)
	reviewRepo := repositories.NewReviewRepository(db)
	mechanicRepo := repositories.NewMechanicRepository(db)
	installRepo := repositories.NewInstallationRepository(db)
	notifRepo := repositories.NewNotificationRepository(db)
	questionRepo := repositories.NewQuestionRepository(db)
	diagramRepo := repositories.NewDiagramRepository(db)

	authSvc := services.NewAuthService(userRepo, jwt, services.AuthConfig{
		LockoutAttempts: cfg.RateLimit.LockoutAttempts,
		LockoutDuration: cfg.RateLimit.LockoutDuration,
	}, log)
	productSvc := services.NewProductService(productRepo, categoryRepo, compatRepo, db)
	categorySvc := services.NewCategoryService(categoryRepo, db)
	cartSvc := services.NewCartService(cartRepo, productRepo, db)
	orderSvc := services.NewOrderService(orderRepo, cartRepo, addressRepo, productRepo, db)
	userSvc := services.NewUserService(userRepo, addressRepo, db)
	wishlistSvc := services.NewWishlistService(wishlistRepo, db)
	reviewSvc := services.NewReviewService(reviewRepo, orderRepo, productRepo, db)
	emailSender := email.NewSender(cfg.Email)
	notifSvc := services.NewNotificationService(notifRepo, userRepo, cache.Client, emailSender, cfg, log)
	notifier := services.NewNotifier(notifSvc, cfg.App.FrontendURL)
	mechanicSvc := services.NewMechanicService(mechanicRepo, installRepo, userRepo, notifier, log, db)
	payoutSvc := services.NewMechanicPayoutService(cfg.Paystack, mechanicRepo, userRepo, log)
	paymentSvc := services.NewPaymentService(cfg.Paystack, orderRepo, installRepo, mechanicRepo, userRepo, db, log)
	installSvc := services.NewInstallationService(installRepo, orderRepo, productRepo, mechanicRepo, paymentSvc, notifier, log, db)
	questionSvc := services.NewQuestionService(questionRepo, productRepo, categoryRepo, notifier, db)
	diagramSvc := services.NewDiagramService(diagramRepo, productRepo, db)

	authH := handlers.NewAuthHandler(authSvc)
	productH := handlers.NewProductHandler(productSvc)
	categoryH := handlers.NewCategoryHandler(categorySvc, productSvc)
	cartH := handlers.NewCartHandler(cartSvc)
	orderH := handlers.NewOrderHandler(orderSvc)
	userH := handlers.NewUserHandler(userSvc)
	wishlistH := handlers.NewWishlistHandler(wishlistSvc)
	reviewH := handlers.NewReviewHandler(reviewSvc)
	mechanicH := handlers.NewMechanicHandler(mechanicSvc, payoutSvc)
	installH := handlers.NewInstallationHandler(installSvc)
	paymentH := handlers.NewPaymentHandler(paymentSvc, payoutSvc)
	notifH := handlers.NewNotificationHandler(notifSvc)
	questionH := handlers.NewQuestionHandler(questionSvc)
	diagramH := handlers.NewDiagramHandler(diagramSvc)

	var store storage.Storage
	if s3Store, err := storage.NewS3(storage.S3Config{
		Bucket:    cfg.S3.Bucket,
		Region:    cfg.S3.Region,
		AccessKey: cfg.S3.AccessKey,
		SecretKey: cfg.S3.SecretKey,
		Endpoint:  cfg.S3.Endpoint,
		PublicURL: cfg.S3.PublicURL,
	}); err == nil && s3Store != nil {
		store = s3Store
	}
	uploadH := handlers.NewUploadHandler(store, cfg.Upload.AllowedTypes, cfg.Upload.MaxSize)
	partIDSvc := services.NewPartIdentificationService(diagramRepo, diagramSvc, store, db)
	partIDH := handlers.NewPartIdentificationHandler(partIDSvc)

	api := r.Group("/api/v1")
	{
		authGroup := api.Group("/auth")
		authGroup.Use(authLimiter.RateLimit())
		{
			authGroup.POST("/register", authH.Register)
			authGroup.POST("/login", authH.Login)
			authGroup.POST("/forgot-password", authH.ForgotPassword)
			authGroup.POST("/reset-password", authH.ResetPassword)
			authGroup.POST("/verify-email", authH.VerifyEmail)
			authGroup.POST("/refresh", authH.Refresh)
			authGroup.POST("/logout", middleware.AuthRequired(jwt, db), authH.Logout)
		}

		api.Use(generalLimiter.RateLimit())
		api.GET("/products", productH.List)
		api.GET("/products/search", productH.Search)
		api.GET("/products/:id", productH.Get)
		api.GET("/products/:id/compatibility", productH.GetCompatibility)
		api.GET("/products/:id/reviews", reviewH.GetByProductID)
		api.GET("/products/:id/questions", questionH.ListByProductID)
		api.GET("/questions", questionH.List)
		api.GET("/questions/:slug", questionH.GetBySlug)
		api.GET("/categories", categoryH.List)
		api.GET("/categories/:id", categoryH.Get)
		api.GET("/categories/:id/products", categoryH.GetProducts)
		api.GET("/mechanics", mechanicH.ListVerified)
		api.GET("/mechanics/:id", mechanicH.GetPublicProfile)
		api.GET("/installation/job-types", installH.ListJobTypes)
		api.GET("/vehicle-systems", diagramH.ListVehicleSystems)
		api.GET("/diagrams", diagramH.List)
		api.GET("/diagrams/:id", diagramH.Get)
		api.GET("/diagrams/:id/hotspots", diagramH.ListHotspots)
		api.GET("/diagrams/:id/hotspots/:hotspotId/products", diagramH.HotspotProducts)

		protected := api.Group("")
		protected.Use(middleware.AuthRequired(jwt, db))
		{
			protected.POST("/products/:id/reviews", reviewH.Create)
			protected.POST("/questions", questionH.Create)
			protected.POST("/questions/:id/answers", middleware.RequireVerifiedMechanic(db), questionH.PostAnswer)
			protected.PATCH("/questions/:id/accept-answer/:answerId", questionH.AcceptAnswer)
			protected.PATCH("/questions/:id/close", questionH.Close)
			protected.POST("/part-identification", partIDH.Identify)
			protected.GET("/cart", cartH.Get)
			protected.POST("/cart/items", cartH.AddItem)
			protected.PUT("/cart/items/:id", cartH.UpdateItem)
			protected.DELETE("/cart/items/:id", cartH.RemoveItem)
			protected.DELETE("/cart", cartH.Clear)
			protected.POST("/orders", orderH.Create)
			protected.GET("/orders", orderH.List)
			protected.GET("/orders/:id", orderH.Get)
			protected.POST("/orders/:id/pay", paymentH.InitializeOrderPayment)
			protected.POST("/orders/:id/refund", paymentH.RefundOrder)
			protected.PUT("/orders/:id/cancel", orderH.Cancel)
			protected.GET("/payments/verify", paymentH.VerifyPayment)
			protected.GET("/payments/banks", paymentH.ListBanks)
			protected.GET("/users/me", userH.GetProfile)
			protected.PUT("/users/me", userH.UpdateProfile)
			protected.PATCH("/users/me", userH.UpdateProfile)
			protected.GET("/users/me/addresses", userH.ListAddresses)
			protected.POST("/users/me/addresses", userH.AddAddress)
			protected.PUT("/users/me/addresses/:id", userH.UpdateAddress)
			protected.DELETE("/users/me/addresses/:id", userH.DeleteAddress)
			protected.GET("/wishlist", wishlistH.Get)
			protected.POST("/wishlist", wishlistH.Add)
			protected.DELETE("/wishlist/:productId", wishlistH.Remove)
			protected.GET("/notifications", notifH.List)
			protected.GET("/notifications/unread-count", notifH.UnreadCount)
			protected.PATCH("/notifications/:id/read", notifH.MarkRead)
			protected.PATCH("/notifications/read-all", notifH.MarkAllRead)
			protected.GET("/users/me/notification-preferences", notifH.GetPreferences)
			protected.PUT("/users/me/notification-preferences", notifH.UpdatePreferences)

			installRoutes := protected.Group("/installation")
			{
				installRoutes.POST("/quotes", installH.CreateQuote)
				installRoutes.GET("/quotes", installH.ListQuotes)
				installRoutes.GET("/quotes/:id", installH.GetQuote)
				installRoutes.POST("/bookings", installH.CreateBooking)
				installRoutes.GET("/bookings", installH.ListBookings)
				installRoutes.GET("/bookings/:id", installH.GetBooking)
				installRoutes.POST("/bookings/:id/pay", paymentH.InitializeBookingPayment)
				installRoutes.POST("/bookings/:id/refund", paymentH.RefundBooking)
				installRoutes.PATCH("/bookings/:id/cancel", installH.CancelBooking)
			}

			mechanicRoutes := protected.Group("/mechanic")
			{
				mechanicRoutes.POST("/apply", mechanicH.Apply)
				mechanicRoutes.GET("/profile", mechanicH.GetMyProfile)
				mechanicRoutes.PUT("/profile", mechanicH.UpdateMyProfile)
				mechanicRoutes.POST("/documents", mechanicH.AddDocument)
				mechanicRoutes.DELETE("/documents/:id", mechanicH.RemoveDocument)
				mechanicRoutes.GET("/payout", mechanicH.GetPayoutStatus)
				mechanicRoutes.POST("/payout", mechanicH.SetupPayout)
			}

			mechanicVerified := protected.Group("/mechanic")
			mechanicVerified.Use(middleware.RequireVerifiedMechanic(db))
			{
				mechanicVerified.GET("/installation/quotes", installH.ListMechanicQuoteLines)
				mechanicVerified.PATCH("/installation/quotes/:id", installH.RespondToQuoteLine)
				mechanicVerified.PUT("/installation/services", installH.SetMechanicInstallServices)
				mechanicVerified.GET("/installation/bookings", installH.ListMechanicBookings)
				mechanicVerified.GET("/installation/bookings/:id", installH.GetMechanicBooking)
				mechanicVerified.PATCH("/installation/bookings/:id/status", installH.UpdateMechanicBookingStatus)
			}
		}

		adminProducts := api.Group("")
		adminProducts.Use(middleware.AuthRequired(jwt, db), middleware.RequireRole(models.RoleAdmin, models.RoleVendor))
		{
			adminProducts.POST("/upload/images", uploadH.UploadImages)
			adminProducts.POST("/products/batch", productH.CreateBatch)
			adminProducts.POST("/products", productH.Create)
			adminProducts.PUT("/products/:id", productH.Update)
			adminProducts.POST("/products/:id/images", productH.AddImages)
			adminProducts.DELETE("/products/:id/images/:imageId", productH.DeleteProductImage)
			adminProducts.POST("/products/:id/compatibility", productH.AddCompatibilities)
			adminProducts.POST("/diagrams", diagramH.Create)
			adminProducts.PUT("/diagrams/:id", diagramH.Update)
			adminProducts.POST("/diagrams/:id/hotspots", diagramH.CreateHotspot)
			adminProducts.PUT("/diagrams/:id/hotspots/:hotspotId", diagramH.UpdateHotspot)
			adminProducts.POST("/diagrams/:id/hotspots/:hotspotId/products", diagramH.LinkProduct)
			adminProducts.DELETE("/diagrams/:id/hotspots/:hotspotId/products/:productId", diagramH.UnlinkProduct)
		}
		adminOnly := api.Group("")
		adminOnly.Use(middleware.AuthRequired(jwt, db), middleware.RequireRole(models.RoleAdmin))
		{
			adminOnly.DELETE("/products/:id", productH.Delete)
			adminOnly.DELETE("/diagrams/:id", diagramH.Delete)
			adminOnly.DELETE("/diagrams/:id/hotspots/:hotspotId", diagramH.DeleteHotspot)
			adminOnly.POST("/categories", categoryH.Create)
			adminOnly.PUT("/categories/:id", categoryH.Update)
			adminOnly.DELETE("/categories/:id", categoryH.Delete)
			adminOnly.GET("/admin/orders", orderH.ListAll)
			adminOnly.PUT("/admin/orders/:id/status", orderH.UpdateStatus)
			adminOnly.POST("/admin/orders/:id/refund", paymentH.AdminRefundOrder)
			adminOnly.POST("/admin/installation/bookings/:id/refund", paymentH.AdminRefundBooking)
			adminOnly.PUT("/admin/users/:id/role", userH.UpdateRole)
			adminOnly.GET("/admin/mechanics", mechanicH.ListAdmin)
			adminOnly.PUT("/admin/mechanics/:userId/verify", mechanicH.Verify)
			adminOnly.PUT("/admin/mechanics/:userId/suspend", mechanicH.Suspend)
			adminOnly.PUT("/admin/mechanics/:userId/reject", mechanicH.Reject)
		}
	}

	r.POST("/webhooks/paystack", paymentH.PaystackWebhook)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/docs", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, "/docs/index.html") })
	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/docs/doc.json")))

	return r
}
