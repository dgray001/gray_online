package main

import (
	"net/http"

	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

const (
	maxPerfBodyBytes = 256 << 10
	maxPerfEntries   = 200
)

func registerPerfRoutes(api *gin.RouterGroup) {
	perf := api.Group("/dev/perf")
	perf.POST("/enable", func(c *gin.Context) {
		if util.PerfToStdout() {
			c.JSON(200, successResponse(true))
			return
		}
		if err := util.SetPerfEnabled(true); err != nil {
			c.JSON(200, failureResponse(err.Error()))
			return
		}
		c.JSON(200, successResponse(true))
	})
	perf.POST("/log", func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxPerfBodyBytes)
		var entries []map[string]any
		if err := c.ShouldBindJSON(&entries); err != nil || len(entries) > maxPerfEntries {
			c.JSON(200, failureResponse("invalid perf log"))
			return
		}
		for _, entry := range entries {
			entry["source"] = "frontend"
			util.PerfWrite(entry)
		}
		c.JSON(200, successResponse(true))
	})
}
