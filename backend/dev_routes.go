package main

import (
	"encoding/json"
	"io"
	"path/filepath"

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
		name := filepath.Base(filepath.Clean(c.Param("name")))
		doc, err := risq.ReadCustomMap(name)
		if err != nil {
			c.JSON(200, failureResponse(err.Error()))
			return
		}
		c.JSON(200, successResponse(json.RawMessage(doc)))
	})
	dev.POST("/maps/:name", func(c *gin.Context) {
		name := filepath.Base(filepath.Clean(c.Param("name")))
		doc, _ := io.ReadAll(c.Request.Body)
		if err := risq.SaveCustomMap(name, doc); err != nil {
			c.JSON(200, failureResponse(err.Error()))
			return
		}
		c.JSON(200, successResponse(true))
	})
	dev.POST("/maps/:name/delete", func(c *gin.Context) {
		name := filepath.Base(filepath.Clean(c.Param("name")))
		if err := risq.DeleteCustomMap(name); err != nil {
			c.JSON(200, failureResponse(err.Error()))
			return
		}
		c.JSON(200, successResponse(true))
	})
	// The optional config_dir segment picks a map's own config folder; without it the default tables are used
	for _, config_dir_segment := range []string{"", "/:config_dir"} {
		dev.GET("/terrains"+config_dir_segment, func(c *gin.Context) {
			configDir := c.Param("config_dir")
			if configDir != "" {
				configDir = filepath.Base(filepath.Clean(configDir))
			}
			config, err := risq.ConfigFromDir(configDir)
			if err != nil {
				c.JSON(200, failureResponse(err.Error()))
				return
			}
			c.JSON(200, successResponse(risq.AllTerrainConfigsToFrontend(config)))
		})
		dev.GET("/entities"+config_dir_segment, func(c *gin.Context) {
			configDir := c.Param("config_dir")
			if configDir != "" {
				configDir = filepath.Base(filepath.Clean(configDir))
			}
			config, err := risq.ConfigFromDir(configDir)
			if err != nil {
				c.JSON(200, failureResponse(err.Error()))
				return
			}
			c.JSON(200, successResponse(risq.AllEntityConfigsToFrontend(config)))
		})
	}
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
