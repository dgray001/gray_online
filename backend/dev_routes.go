package main

import (
	"encoding/json"
	"io"

	"github.com/dgray001/gray_online/game/games/risq"
	"github.com/gin-gonic/gin"
)

func registerRisqDevRoutes(api *gin.RouterGroup) {
	dev := api.Group("/dev/risq")
	dev.GET("/maps", func(c *gin.Context) {
		names, err := risq.ListCustomMaps()
		if err != nil {
			c.JSON(200, failureResponse(err.Error()))
			return
		}
		c.JSON(200, successResponse(names))
	})
	dev.GET("/maps/:name", func(c *gin.Context) {
		doc, err := risq.ReadCustomMap(c.Param("name"))
		if err != nil {
			c.JSON(200, failureResponse(err.Error()))
			return
		}
		c.JSON(200, successResponse(json.RawMessage(doc)))
	})
	dev.POST("/maps/:name", func(c *gin.Context) {
		doc, _ := io.ReadAll(c.Request.Body)
		if err := risq.SaveCustomMap(c.Param("name"), doc); err != nil {
			c.JSON(200, failureResponse(err.Error()))
			return
		}
		c.JSON(200, successResponse(true))
	})
	dev.POST("/maps/:name/delete", func(c *gin.Context) {
		if err := risq.DeleteCustomMap(c.Param("name")); err != nil {
			c.JSON(200, failureResponse(err.Error()))
			return
		}
		c.JSON(200, successResponse(true))
	})
	dev.GET("/terrains", func(c *gin.Context) {
		c.JSON(200, successResponse(risq.AllTerrainConfigsToFrontend()))
	})
	dev.POST("/preview", func(c *gin.Context) {
		doc, _ := io.ReadAll(c.Request.Body)
		snapshot, err := risq.PreviewCustomMap(doc)
		if err != nil {
			c.JSON(200, failureResponse(err.Error()))
			return
		}
		c.JSON(200, successResponse(snapshot))
	})
}
