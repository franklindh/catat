package util

import "github.com/gin-gonic/gin"

func ErrorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}

func ErrorResponseWithMessage(message string) gin.H {
	return gin.H{"error": message}
}
