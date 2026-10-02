package bootstrap

import (
	businessapp "QueueLite/internal/business/app"
	businesshttp "QueueLite/internal/business/inbound/http"
	businessoutbound "QueueLite/internal/business/outbound"
	counterapp "QueueLite/internal/counter/app"
	counterhttp "QueueLite/internal/counter/inbound/http"
	counteroutbound "QueueLite/internal/counter/outbound"
	"QueueLite/internal/middleware"
	"QueueLite/internal/operations"
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
	businessRepo := businessoutbound.NewBusinessRepo(db)
	counterRepo := counteroutbound.NewCounterRepo(db)
	go queueapp.RunDatabaseWorkerStream(ctx, rdb, queueRepo)
	queueService := queueapp.NewQueueService(queueRepo, subscriptionService, rdb, businessRepo, userRepo, counterRepo)
	queueHandler := queuehttp.NewQueueHandler(queueService)

	counterService := counterapp.NewCounterService(counterRepo, queueRepo, subscriptionService, rdb)
	counterHandler := counterhttp.NewCounterHandler(counterService)
	operationsHandler := operations.New(businessRepo, counterRepo, queueRepo, rdb)

	businessService := businessapp.NewBusinessService(businessRepo, subscriptionService, counterService)
	businessHandler := businesshttp.NewBusinessHandler(businessService)

	router.Route("/users", func(r chi.Router) {
		r.Post("/logout", userHandler.LogoutUser)
		r.Group(func(public chi.Router) {
			public.Use(middleware.PublicMiddleware(userRepo))
			public.Post("/", userHandler.RegisterUser)
			public.Post("/login", userHandler.LoginUser)
		})
		r.Group(func(private chi.Router) {
			private.Use(middleware.AuthMiddleware(userRepo))
			private.Get("/me", userHandler.GetCurrentUser)
			private.Get("/{userID}", userHandler.GetUser)
			private.With(middleware.RequireSelfUser).Put("/{userID}", userHandler.UpdateUser)
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
			private.Get("/mine", businessHandler.GetMyBusinesses)
			private.Get("/{businessID}/counters", operationsHandler.ListCounters)
			private.Get("/{businessID}/members", operationsHandler.ListMembers)
			private.With(operationsHandler.RequireBusiness).Get("/{businessID}/queues", queueHandler.GetAllQueueByBusiness)
			private.With(businessHandler.RequireManagement).Put("/{businessID}", businessHandler.UpdateBusiness)
			private.With(businessHandler.RequireManagement).Delete("/{businessID}", businessHandler.DeleteBusiness)
		})
	})

	router.Route("/queues", func(r chi.Router) {
		r.With(middleware.AuthMiddleware(userRepo), operationsHandler.RequireBusiness).Get("/business/{businessID}", queueHandler.GetAllQueueByBusiness)
		r.With(middleware.AuthMiddleware(userRepo)).Get("/me", queueHandler.GetMyQueues)
		r.Group(func(public chi.Router) {
			public.Use(middleware.PublicMiddleware(userRepo))
			public.Get("/qr/{businessID}/resolve", queueHandler.ResolveQueueQR)
			public.Post("/qr/{businessID}", queueHandler.RegisterQueueByQr)
			public.Post("/business/{businessID}/join", queueHandler.JoinBusinessQueue)
			public.Post("/", queueHandler.RegisterQueue)
			// public.Get("/business/{businessID}/summary", queueHandler.GetBusinessPublicQueueSummary)
			// public.Get("/business/{businessID}/state/{state}", queueHandler.GetAllQueueByBusinessFilterState)
		})
		r.Group(func(protected chi.Router) {
			protected.Use(middleware.PublicMiddleware(userRepo))
			protected.Use(middleware.ProtectedMiddleware(*queueService))
			protected.Get("/{queueID}", queueHandler.GetQueue)
			protected.Get("/{queueID}/state", queueHandler.GetQueueState)
			protected.Get("/{queueID}/customer-status", queueHandler.GetCustomerQueueStatus)
			protected.Delete("/{queueID}", queueHandler.DeleteQueue)
			protected.Put("/{queueID}", queueHandler.UpdateQueue)
			protected.Patch("/{queueID}/state", queueHandler.UpdateState)
			protected.Patch("/{queueID}/done", queueHandler.MarkAsDone)
		})
	})

	router.Route("/counters", func(r chi.Router) {
		r.Group(func(private chi.Router) {
			private.Use(middleware.AuthMiddleware(userRepo))
			private.With(operationsHandler.RequireCreate).Post("/", counterHandler.CreateCounter)
			private.Group(func(authorized chi.Router) {
				authorized.Use(operationsHandler.RequireCounter)
				authorized.Get("/{counterID}", counterHandler.GetCounter)
				authorized.Post("/{counterID}/business/{businessID}/call-next", counterHandler.CallNextQueue)
				authorized.Post("/{counterID}/queues/{queueID}/process", counterHandler.ProcessCalledQueue)
				authorized.Post("/{counterID}/queues/{queueID}/skip", counterHandler.SkipQueue)
				authorized.Delete("/{counterID}/queues/{queueID}", counterHandler.RemoveQueueFromCounter)
				authorized.Put("/{counterID}", counterHandler.UpdateCounter)
				authorized.Delete("/{counterID}", counterHandler.DeleteCounter)
			})
		})
	})

	router.Route("/subscriptions", func(r chi.Router) {
		r.Group(func(private chi.Router) {
			private.Use(middleware.AuthMiddleware(userRepo))
			subscriptionhttp.RegisterAccountRoutes(private, subscriptionHandler, businessHandler.RequireManagement, businessRepo)
		})
	})

	return router
}
