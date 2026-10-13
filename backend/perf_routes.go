package main

import (
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

func registerPerfRoutes(api *gin.RouterGroup) {
	perf := api.Group("/dev/perf")
	perf.POST("/enable", func(c *gin.Context) {
		if err := util.SetPerfEnabled(true); err != nil {
			c.JSON(200, failureResponse(err.Error()))
			return
		}
		c.JSON(200, successResponse(true))
	})
	perf.POST("/log", func(c *gin.Context) {
		var entries []map[string]any
		if err := c.ShouldBindJSON(&entries); err != nil {
			c.JSON(200, failureResponse(err.Error()))
			return
		}
		for _, entry := range entries {
			entry["source"] = "frontend"
			util.PerfWrite(entry)
		}
		c.JSON(200, successResponse(true))
	})
}
