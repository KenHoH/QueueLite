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
	go queueapp.RunDatabaseWorkerStream(ctx, rdb, queueRepo)
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
			public.Post("/", userHandler.RegisterUser)
			public.Post("/login", userHandler.LoginUser)
		})
		r.Group(func(private chi.Router) {
			private.Use(middleware.AuthMiddleware(userRepo))
			private.Get("/{userID}", userHandler.GetUser)
			private.Put("/{userID}", userHandler.UpdateUser)
		})
	})

	router.Route("/businesses", func(r chi.Router) {
		r.Group(func(public chi.Router) {
			public.Use(middleware.PublicMiddleware(userRepo))
			public.Get("/", businessHandler.GetBusinessAll)
			public.Get("/search", businessHandler.SearchBusiness)
			public.Get("/{businessID}", businessHandler.GetBusiness)
		})
		r.Group(func(private chi.Router) {
			private.Use(middleware.AuthMiddleware(userRepo))
			private.Post("/", businessHandler.CreateBusiness)
			private.Put("/{businessID}", businessHandler.UpdateBusiness)
			private.Delete("/{businessID}", businessHandler.DeleteBusiness)
		})
	})

	router.Route("/queues", func(r chi.Router) {
		r.Group(func(public chi.Router) {
			public.Use(middleware.PublicMiddleware(userRepo))
			public.Put("/{queueID}", queueHandler.UpdateQueue)
			public.Post("/qr/{businessID}", queueHandler.RegisterQueueByQr)
			public.Patch("/{queueID}/state", queueHandler.UpdateState)
			public.Patch("/{queueID}/done", queueHandler.MarkAsDone)
			public.Post("/", queueHandler.RegisterQueue)
			public.Get("/business/{businessID}", queueHandler.GetAllQueueByBusiness)
			// public.Get("/business/{businessID}/summary", queueHandler.GetBusinessPublicQueueSummary)
			// public.Get("/business/{businessID}/state/{state}", queueHandler.GetAllQueueByBusinessFilterState)
		})
		r.Group(func(protected chi.Router) {
			protected.Use(middleware.ProtectedMiddleware(*queueService))
			protected.Get("/{queueID}", queueHandler.GetQueue)
			protected.Get("/{queueID}/state", queueHandler.GetQueueState)
			protected.Delete("/{queueID}", queueHandler.DeleteQueue)
		})
	})

	router.Route("/counters", func(r chi.Router) {
		r.Group(func(public chi.Router) {
			public.Use(middleware.PublicMiddleware(userRepo))
			public.Get("/{counterID}", counterHandler.GetCounter)
		})
		r.Group(func(private chi.Router) {
			private.Use(middleware.AuthMiddleware(userRepo))
			private.Post("/", counterHandler.CreateCounter)
			private.Post("/{counterID}/business/{businessID}/call-next", counterHandler.CallNextQueue)
			private.Post("/{counterID}/queues/{queueID}/process", counterHandler.ProcessCalledQueue)
			private.Post("/{counterID}/queues/{queueID}/skip", counterHandler.SkipQueue)
			private.Delete("/{counterID}/queues/{queueID}", counterHandler.RemoveQueueFromCounter)
			private.Put("/{counterID}", counterHandler.UpdateCounter)
			private.Delete("/{counterID}", counterHandler.DeleteCounter)
		})
	})

	router.Route("/subscriptions", func(r chi.Router) {
		r.Group(func(private chi.Router) {
			private.Use(middleware.AuthMiddleware(userRepo))
			private.Get("/", subscriptionHandler.GetAllSubscription)
			private.Get("/businesses/{businessID}", subscriptionHandler.GetBusinessSubscriptionInfo)
			private.Get("/users/{userID}", subscriptionHandler.GetUserSubscriptionInfo)
			private.Post("/users/{userID}/use", subscriptionHandler.UseUserSubscription)
			private.Patch("/users/{userID}/slots", subscriptionHandler.AddUserSlot)
			private.Patch("/businesses/{businessID}/capacity/decrease", subscriptionHandler.DecreaseBusinessCapacity)
			private.Get("/{subscriptionID}", subscriptionHandler.GetSubscription)
			private.Put("/{subscriptionID}", subscriptionHandler.UpdateSubscription)
			private.Patch("/{subscriptionID}/time", subscriptionHandler.UpdateSubscriptionTime)
			private.Patch("/{subscriptionID}/activate", subscriptionHandler.ActivateUserSubscription)
			private.Patch("/{subscriptionID}/deactivate", subscriptionHandler.DeactivateUserSubscription)
			private.Delete("/{subscriptionID}", subscriptionHandler.DeleteSubscription)
		})
	})

	return router
}
