package router

import (
	"9itchi-account-api/handlers"
	"9itchi-account-api/services"

	"github.com/gin-gonic/gin"
)

func InitializeRouter(services *services.Services) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	initializeMiddlewares(r)
	initializeRoutes(r, services)

	return r
}

func initializeRoutes(r *gin.Engine, services *services.Services) {
	// Health endpoints
	r.GET("/", handlers.GetHealth)
	r.GET("/health", handlers.GetHealth)

	// public endpoints
	{
		r.POST("/auth/signup", handlers.Signup(services.Auth))
		r.POST("/auth/login", handlers.Login(services.Auth))
		r.POST("/auth/refresh", handlers.RefreshToken(services.Auth))
		r.POST("/auth/forgot-password", handlers.ForgotPassword(services.Auth))
		r.POST("/auth/reset-password", handlers.ResetPassword(services.Auth))
		r.GET("/auth/google", handlers.GoogleSignIn(services.Auth))
		r.GET("/auth/google/callback", handlers.GoogleCallback(services.Auth))
		r.GET("/auth/linkedin", handlers.LinkedInSignIn(services.Auth))
		r.GET("/auth/linkedin/callback", handlers.LinkedInCallback(services.Auth))
	}

	// user endpoints
	userGroup := r.Group("/")
	userGroup.Use(AuthMiddleware())
	{
		userGroup.POST("/auth/logout", handlers.Logout(services.Auth))
		userGroup.GET("/users/me", handlers.GetMe(services.User))
		userGroup.PUT("/users/me", handlers.UpdateMe(services.User))
		userGroup.PUT("/users/me/language", handlers.SetLanguage(services.User))
		userGroup.DELETE("/users/me", handlers.DeleteMe(services.User))
	}

	// admin endpoints
	adminGroup := r.Group("/")
	adminGroup.Use(AuthMiddleware(), RequireRole("admin"))
	{
		adminGroup.GET("/admin/users", handlers.ListUsers(services.User))
		adminGroup.GET("/admin/users/:id", handlers.GetUser(services.User))
		adminGroup.PUT("/admin/users/:id/suspend", handlers.SuspendUser(services.User))
		adminGroup.DELETE("/admin/users/:id", handlers.DeleteUser(services.User))
	}

	// internal endpoints
	internalGroup := r.Group("/")
	internalGroup.Use(InternalAPIKeyMiddleware())
	{
		internalGroup.GET("/internal/users/:id", handlers.InternalGetUser(services.User))
	}

	// Fallback
	r.NoRoute(handlers.NoRoute)
}
