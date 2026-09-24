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

	userRepo := useroutbound.NewUserRepo(db)
	router.Use(middleware.AuthMiddleware(userRepo))
	userService := userapp.NewUserService(userRepo)
	userHandler := userhttp.NewUserHandler(userService)

	businessRepo := businessoutbound.NewBusinessRepo(db)
	businessService := businessapp.NewBusinessService(businessRepo)
	businessHandler := businesshttp.NewBusinessHandler(businessService)

	queueRepo := queueoutbound.NewQueueRepo(db)
	queueService := queueapp.NewQueueService(queueRepo)
	queueHandler := queuehttp.NewQueueHandler(queueService)

	counterRepo := counteroutbound.NewCounterRepo(db)
	counterService := counterapp.NewCounterService(counterRepo, queueRepo)
	counterHandler := counterhttp.NewCounterHandler(counterService)

	subscriptionRepo := subscriptionoutbound.NewSubscriptionRepo(db)
	subscriptionService := subscriptionapp.NewSubscriptionService(subscriptionRepo)
	subscriptionHandler := subscriptionhttp.NewSubscriptionHandler(subscriptionService)

	router.Mount("/users", userHandler.Routes())
	router.Mount("/businesses", businessHandler.Routes())
	router.Mount("/queues", queueHandler.Routes())
	router.Mount("/counters", counterHandler.Routes())
	router.Mount("/subscriptions", subscriptionHandler.Routes())

	return router
}
