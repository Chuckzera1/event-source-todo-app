package main

import (
	"log"
	"os"

	"github.com/Chuckzera1/event-source-todo-app/internal/di"
	"github.com/Chuckzera1/event-source-todo-app/internal/infrastructure"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	loadEnv()

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	db, err := infrastructure.NewGorm(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	createEventHandler := di.NewCreateEventHandlerDI(db)
	r.POST("/events", createEventHandler.Handle)
}

func loadEnv() {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("warning: .env not loaded: %v", err)
	}
}
