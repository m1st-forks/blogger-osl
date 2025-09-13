package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	err := godotenv.Load() // 👈 load .env file
	if err != nil {
		log.Fatal(err)
	}

	if v := os.Getenv("POSTS_JSON"); v != "" {
		postsFilePath = v
	}
	if abs, err := os.Getwd(); err == nil {
		_ = abs
	}
	if err := loadFromDisk(); err != nil {
		log.Printf("warning: failed to load posts from %s: %v", postsFilePath, err)
	}

	if v := os.Getenv("POSTS_DIR"); v != "" {
		postsDir = v
	}
	_ = os.MkdirAll(postsDir, 0o755)

	if v := os.Getenv("THUMBS_DIR"); v != "" {
		thumbsDir = v
	}
	_ = os.MkdirAll(thumbsDir, 0o755)

	api := r.Group("/api")
	{
		api.GET("/posts", listPosts)
		api.GET("/posts/:id", getPost)

		api.POST("/posts", requireValidator, requireAllowedUser, createPost)
		api.PATCH("/posts/:id", requireValidator, requireAllowedUser, patchPost)
		api.DELETE("/posts/:id", requireValidator, requireAllowedUser, deletePost)

		api.GET("/thumbnails", listThumbnails)
		api.POST("/thumbnails", requireValidator, requireAllowedUser, uploadThumbnail)
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "../static"
	}
	r.Static("/static", staticDir)
	r.Static("/thumbnails", thumbsDir)
	r.GET("/", func(c *gin.Context) {
		c.File(filepath.Join(staticDir, "index.html"))
	})
	r.GET("/admin", func(c *gin.Context) {
		c.File(filepath.Join(staticDir, "admin.html"))
	})
	r.GET("/create", func(c *gin.Context) {
		c.Redirect(302, "/admin")
	})
	r.GET("/post/:id", func(c *gin.Context) {
		c.File(filepath.Join(staticDir, "post.html"))
	})
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if p == "/api" || strings.HasPrefix(p, "/api/") {
			c.AbortWithStatus(404)
			return
		}
		// For unknown paths, send home page
		c.File(filepath.Join(staticDir, "index.html"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
