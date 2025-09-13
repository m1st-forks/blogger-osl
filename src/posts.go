package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func createPost(c *gin.Context) {
	var in createPostInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}
	// Require validated user and set as author (server-trusted)
	if ifUser, ok := validatedUser(c); ok {
		in.Author = ifUser
	} else {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "missing validator"})
		return
	}
	if in.Author == "" || in.Title == "" {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "author and title are required"})
		return
	}
	// optional: thumbnail is a path like /thumbnails/...

	postsMu.Lock()
	id := nextID
	nextID++
	p := &Post{
		ID:          id,
		Author:      in.Author,
		Title:       in.Title,
		Description: in.Description,
		Timestamp:   time.Now().UnixMilli(),
		Thumbnail:   in.Thumbnail,
	}
	posts[id] = p
	if err := writeContentLocked(id, in.Content); err != nil {
		delete(posts, id)
		nextID--
		postsMu.Unlock()
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to write content"})
		return
	}
	if err := saveToDiskLocked(); err != nil {
		delete(posts, id)
		_ = deleteContentLocked(id)
		nextID--
		postsMu.Unlock()
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to persist post"})
		return
	}
	postsMu.Unlock()

	c.JSON(http.StatusCreated, p)
}

func listPosts(c *gin.Context) {
	postsMu.RLock()
	defer postsMu.RUnlock()

	out := make([]Post, 0, len(posts))
	for _, p := range posts {
		out = append(out, *p)
	}
	c.JSON(http.StatusOK, out)
}

func getPost(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	postsMu.RLock()
	p, exists := posts[id]
	postsMu.RUnlock()
	if !exists {
		c.JSON(http.StatusNotFound, errorResponse{Error: "post not found"})
		return
	}
	content, err := readContent(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to read content"})
		return
	}
	resp := struct {
		*Post
		Content string `json:"content"`
	}{Post: p, Content: content}
	c.JSON(http.StatusOK, resp)
}

func patchPost(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}
	if len(body) == 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "empty body"})
		return
	}

	postsMu.Lock()
	p, exists := posts[id]
	if !exists {
		postsMu.Unlock()
		c.JSON(http.StatusNotFound, errorResponse{Error: "post not found"})
		return
	}
	original := *p
	var (
		contentProvided bool
		newContent      string
		prevContent     string
		prevContentRead bool
	)
	if v, ok := body["title"]; ok {
		s, ok := v.(string)
		if !ok {
			postsMu.Unlock()
			c.JSON(http.StatusBadRequest, errorResponse{Error: "title must be a string"})
			return
		}
		p.Title = s
	}
	if v, ok := body["description"]; ok {
		s, ok := v.(string)
		if !ok {
			postsMu.Unlock()
			c.JSON(http.StatusBadRequest, errorResponse{Error: "description must be a string"})
			return
		}
		p.Description = s
	}
	// Enforce author cannot be changed unless matches validated user
	if v, ok := body["author"]; ok {
		s, ok := v.(string)
		if !ok {
			postsMu.Unlock()
			c.JSON(http.StatusBadRequest, errorResponse{Error: "author must be a string"})
			return
		}
		if vu, vok := validatedUser(c); vok {
			// only allow set if matches validator username
			if s != vu {
				postsMu.Unlock()
				c.JSON(http.StatusForbidden, errorResponse{Error: "cannot change author"})
				return
			}
			p.Author = s
		} else {
			// no validator: reject changing author
			postsMu.Unlock()
			c.JSON(http.StatusUnauthorized, errorResponse{Error: "missing validator"})
			return
		}
	}
	if v, ok := body["content"]; ok {
		s, ok := v.(string)
		if !ok {
			postsMu.Unlock()
			c.JSON(http.StatusBadRequest, errorResponse{Error: "content must be a string"})
			return
		}
		contentProvided = true
		newContent = s
		if c0, err := readContent(id); err == nil {
			prevContent = c0
			prevContentRead = true
		}
		if err := writeContentLocked(id, newContent); err != nil {
			*p = original
			postsMu.Unlock()
			c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to write content"})
			return
		}
	}
	if v, ok := body["thumbnail"]; ok {
		s, ok := v.(string)
		if !ok {
			postsMu.Unlock()
			c.JSON(http.StatusBadRequest, errorResponse{Error: "thumbnail must be a string"})
			return
		}
		p.Thumbnail = s
	}
	if err := saveToDiskLocked(); err != nil {
		*p = original
		if contentProvided && prevContentRead {
			_ = writeContentLocked(id, prevContent)
		}
		postsMu.Unlock()
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to persist changes"})
		return
	}
	postsMu.Unlock()

	c.JSON(http.StatusOK, p)
}

func deletePost(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	postsMu.Lock()
	p, exists := posts[id]
	if !exists {
		postsMu.Unlock()
		c.JSON(http.StatusNotFound, errorResponse{Error: "post not found"})
		return
	}
	backup := *p
	prevContent, _ := readContent(id)
	delete(posts, id)
	if err := deleteContentLocked(id); err != nil {
		posts[id] = &backup
		postsMu.Unlock()
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to delete content"})
		return
	}
	if err := saveToDiskLocked(); err != nil {
		posts[id] = &backup
		_ = writeContentLocked(id, prevContent)
		postsMu.Unlock()
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to persist deletion"})
		return
	}
	postsMu.Unlock()
	c.Status(http.StatusNoContent)
}

// listThumbnails returns file names under thumbsDir (non-recursive)
func listThumbnails(c *gin.Context) {
	entries, err := os.ReadDir(thumbsDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to read thumbnails"})
		return
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// basic allowlist by extension
		ext := filepath.Ext(name)
		switch ext {
		case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg":
			out = append(out, "/thumbnails/"+name)
		default:
			// ignore others
		}
	}
	c.JSON(http.StatusOK, out)
}

// uploadThumbnail accepts multipart/form-data with file field "file" and writes to thumbsDir
func uploadThumbnail(c *gin.Context) {
	// limit: rely on reverse proxy limits; here, just accept
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "missing file"})
		return
	}
	// sanitize filename: use base only
	name := filepath.Base(file.Filename)
	// basic extension check
	ext := filepath.Ext(name)
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg":
		// ok
	default:
		c.JSON(http.StatusBadRequest, errorResponse{Error: "unsupported file type"})
		return
	}
	// if exists, add suffix
	dst := filepath.Join(thumbsDir, name)
	base := name[:len(name)-len(ext)]
	i := 1
	for {
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			break
		}
		cand := base + "-" + strconv.Itoa(i) + ext
		dst = filepath.Join(thumbsDir, cand)
		i++
		if i > 1000 { // safety
			c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to allocate filename"})
			return
		}
	}
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to save file"})
		return
	}
	// return public path
	c.JSON(http.StatusCreated, gin.H{"path": "/thumbnails/" + filepath.Base(dst)})
}
