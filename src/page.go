package main

import (
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func postPage(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	postsMu.RLock()
	p, exists := posts[id]
	postsMu.RUnlock()
	if !exists {
		c.Status(http.StatusNotFound)
		// serve static 404-ish page
		staticDir := os.Getenv("STATIC_DIR")
		if staticDir == "" {
			staticDir = "../static"
		}
		c.File(filepath.Join(staticDir, "post.html"))
		return
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		// best effort derive
		scheme := "http"
		if strings.EqualFold(c.Request.Header.Get("X-Forwarded-Proto"), "https") || c.Request.TLS != nil {
			scheme = "https"
		}
		host := c.Request.Host
		if host == "" {
			host = "localhost:8080"
		}
		baseURL = fmt.Sprintf("%s://%s", scheme, host)
	}

	pageURL := fmt.Sprintf("%s/post/%d", baseURL, p.ID)
	title := html.EscapeString(p.Title)
	desc := html.EscapeString(p.Description)
	author := html.EscapeString(p.Author)
	image := ""
	if p.Thumbnail != "" {
		// If thumbnail is already absolute, keep it; else make absolute
		if strings.HasPrefix(p.Thumbnail, "http://") || strings.HasPrefix(p.Thumbnail, "https://") {
			image = p.Thumbnail
		} else {
			image = strings.TrimRight(baseURL, "/") + p.Thumbnail
		}
	}

	// Minimal HTML with OG/Twitter tags and a noscript fallback linking to the JS-rendered page
	htmlOut := fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>%s • Warpdrive</title>
  <meta property="og:type" content="article" />
  <meta property="og:url" content="%s" />
  <meta property="og:title" content="%s" />
  <meta property="og:description" content="%s" />
  %s
  <meta name="twitter:card" content="summary_large_image" />
  <meta name="twitter:title" content="%s" />
  <meta name="twitter:description" content="%s" />
  %s
  <meta name="author" content="%s" />
  <meta name="robots" content="index,follow" />
  <link rel="canonical" href="%s" />
  <script>location.replace('%s');</script>
  <noscript>
    <meta http-equiv="refresh" content="0; url=%s" />
  </noscript>
</head>
<body>
  <p>Loading post… If you are not redirected, <a href="%s">click here</a>.</p>
</body>
</html>`,
		title,
		pageURL,
		title,
		desc,
		// og:image
		func() string {
			if image != "" {
				return fmt.Sprintf("<meta property=\"og:image\" content=\"%s\" />", html.EscapeString(image))
			}
			return ""
		}(),
		title,
		desc,
		// twitter:image
		func() string {
			if image != "" {
				return fmt.Sprintf("<meta name=\"twitter:image\" content=\"%s\" />", html.EscapeString(image))
			}
			return ""
		}(),
		author,
		pageURL,
		pageURL,
		pageURL,
		pageURL,
	)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlOut))
}
