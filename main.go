package main

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/saqibullah/smart-disease-predictor-backend/api"
)

func main() {
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})
	r.POST("/predict", api.PredictHandler)
	r.POST("/extract", api.ExtractHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
