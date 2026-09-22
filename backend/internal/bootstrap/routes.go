package bootstrap

import (
	userapp "QueueLite/internal/user/app"
	userhttp "QueueLite/internal/user/inbound/http"
	useroutbound "QueueLite/internal/user/outbound"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB) *chi.Mux {
	router := chi.NewRouter()

	userRepo := useroutbound.NewUserRepo(db)
	userService := userapp.NewUserService(userRepo)
	userHandler := userhttp.NewUserHandler(userService)

	router.Mount("/users", userHandler.Routes())

	return router
}
