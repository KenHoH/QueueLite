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
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, rdb *redis.Client) *chi.Mux {
	router := chi.NewRouter()

	subscriptionRepo := subscriptionoutbound.NewSubscriptionRepo(db)
	subscriptionService := subscriptionapp.NewSubscriptionService(subscriptionRepo)
	subscriptionHandler := subscriptionhttp.NewSubscriptionHandler(subscriptionService)

	userRepo := useroutbound.NewUserRepo(db)
	userService := userapp.NewUserService(userRepo, subscriptionService)
	userHandler := userhttp.NewUserHandler(userService)

	queueRepo := queueoutbound.NewQueueRepo(db)
	queueService := queueapp.NewQueueService(queueRepo, subscriptionService, rdb)
	queueHandler := queuehttp.NewQueueHandler(queueService)

	counterRepo := counteroutbound.NewCounterRepo(db)
	counterService := counterapp.NewCounterService(counterRepo, queueRepo, subscriptionService, rdb)
	counterHandler := counterhttp.NewCounterHandler(counterService)

	businessRepo := businessoutbound.NewBusinessRepo(db)
	businessService := businessapp.NewBusinessService(businessRepo, subscriptionService, counterService)
	businessHandler := businesshttp.NewBusinessHandler(businessService)

	router.Route("/users", func(r chi.Router) {
		r.Group(func(public chi.Router) {
			public.Use(middleware.PublicMiddleware(userRepo))
			public.Mount("/", userHandler.PublicRoutes())
		})
		r.Group(func(private chi.Router) {
			private.Use(middleware.AuthMiddleware(userRepo))
			private.Mount("/", userHandler.PrivateRoutes())
		})
	})

	router.Route("/businesses", func(r chi.Router) {
		r.Group(func(public chi.Router) {
			public.Use(middleware.PublicMiddleware(userRepo))
			public.Mount("/", businessHandler.PublicRoutes())
		})
	})

	router.Route("/queues", func(r chi.Router) {
		r.Group(func(public chi.Router) {
			public.Use(middleware.PublicMiddleware(userRepo))
			public.Mount("/", queueHandler.PublicRoutes())
		})
		r.Group(func(protected chi.Router) {
			protected.Use(middleware.ProtectedMiddleware(*queueService))
			protected.Mount("/", queueHandler.ProtectedRoutes())
		})
	})

	router.Route("/counters", func(r chi.Router) {
		r.Group(func(public chi.Router) {
			public.Use(middleware.PublicMiddleware(userRepo))
			public.Mount("/", counterHandler.PublicRoutes())
		})
		r.Group(func(private chi.Router) {
			private.Use(middleware.AuthMiddleware(userRepo))
			private.Mount("/", counterHandler.PrivateRoutes())
		})
	})

	router.Route("/subscriptions", func(r chi.Router) {
		r.Group(func(private chi.Router) {
			private.Use(middleware.AuthMiddleware(userRepo))
			private.Mount("/", subscriptionHandler.Routes())
		})
	})

	return router
}
