package bootstrap

import (
	businessapp "QueueLite/internal/business/app"
	businesshttp "QueueLite/internal/business/inbound/http"
	businessoutbound "QueueLite/internal/business/outbound"
	counterapp "QueueLite/internal/counter/app"
	counterhttp "QueueLite/internal/counter/inbound/http"
	counteroutbound "QueueLite/internal/counter/outbound"
	"QueueLite/internal/middleware"
	queueapp "QueueLite/internal/queue/app"
	queuehttp "QueueLite/internal/queue/inbound/http"
	queueoutbound "QueueLite/internal/queue/outbound"
	subscriptionapp "QueueLite/internal/subscription/app"
	subscriptionhttp "QueueLite/internal/subscription/inbound/http"
	subscriptionoutbound "QueueLite/internal/subscription/outbound"
	userapp "QueueLite/internal/user/app"
	userhttp "QueueLite/internal/user/inbound/http"
	useroutbound "QueueLite/internal/user/outbound"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB) *chi.Mux {
	router := chi.NewRouter()

	subscriptionRepo := subscriptionoutbound.NewSubscriptionRepo(db)
	subscriptionService := subscriptionapp.NewSubscriptionService(subscriptionRepo)
	subscriptionHandler := subscriptionhttp.NewSubscriptionHandler(subscriptionService)

	userRepo := useroutbound.NewUserRepo(db)
	userService := userapp.NewUserService(userRepo, subscriptionService)
	userHandler := userhttp.NewUserHandler(userService)

	queueRepo := queueoutbound.NewQueueRepo(db)
	queueService := queueapp.NewQueueService(queueRepo, subscriptionService)
	queueHandler := queuehttp.NewQueueHandler(queueService)

	counterRepo := counteroutbound.NewCounterRepo(db)
	counterService := counterapp.NewCounterService(counterRepo, queueRepo, subscriptionService)
	counterHandler := counterhttp.NewCounterHandler(counterService)

	businessRepo := businessoutbound.NewBusinessRepo(db)
	businessService := businessapp.NewBusinessService(businessRepo, subscriptionService, counterService)
	businessHandler := businesshttp.NewBusinessHandler(businessService)

	// private group: requires a real registered user token.
	router.Group(func(private chi.Router) {
		private.Use(middleware.AuthMiddleware(userRepo))
		private.Mount("/users", userHandler.PrivateRoutes())
		private.Mount("/businesses", businessHandler.PublicRoutes())
		private.Mount("/queues", queueHandler.PublicRoutes())
		private.Mount("/counters", counterHandler.PrivateRoutes())
		private.Mount("/subscriptions", subscriptionHandler.Routes())
	})

	// public group: no authentication required.
	router.Group(func(public chi.Router) {
		public.Mount("/users", userHandler.PublicRoutes())
		public.Mount("/businesses", businessHandler.PrivateRoutes())
		public.Mount("/queues", queueHandler.PrivateRoutes())
		public.Mount("/counters", counterHandler.PublicRoutes())
	})

	// protected group: requires a valid queue token.
	router.Group(func(protected chi.Router) {
		protected.Use(middleware.ProtectedMiddleware(*queueService))
		protected.Mount("/queues", queueHandler.ProtectedRoutes())
	})

	return router
}
