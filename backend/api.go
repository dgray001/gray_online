package main

import "github.com/gin-gonic/gin"

func successResponse(result any) gin.H {
	return gin.H{
		"success": true,
		"result":  result,
	}
}

func failureResponse(error_message string) gin.H {
	return gin.H{
		"success":       false,
		"error_message": error_message,
	}
}

func failureResponseWiothResult(error_message string, result any) gin.H {
	return gin.H{
		"success":       false,
		"result":        result,
		"error_message": error_message,
	}
}
