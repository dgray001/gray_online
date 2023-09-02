package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.GET("/ping1", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong1",
		})
	})
	r.GET("/ping2", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong2",
		})
	})
	v2 := r.Group("/group")
	{
		v2.GET("/one", one)
		v2.GET("/two", two)
		v2.GET("/three", three)
	}
	r.Run() // listen and serve on 0.0.0.0:8080
}

func one(c *gin.Context) {
	c.JSON(201, gin.H{
		"message": "1",
	})
}

func two(c *gin.Context) {
	c.JSON(202, gin.H{
		"message": "2",
	})
}

func three(c *gin.Context) {
	c.JSON(203, gin.H{
		"message": "3",
	})
}
