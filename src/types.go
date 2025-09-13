package main

import (
	"sync"
)

type Post struct {
	ID          int64  `json:"id"`
	Author      string `json:"author"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Timestamp   int64  `json:"timestamp"`
	Thumbnail   string `json:"thumbnail"`
}

var (
	postsMu sync.RWMutex
	posts         = map[int64]*Post{}
	nextID  int64 = 1
)

type errorResponse struct {
	Error string `json:"error"`
}

type createPostInput struct {
	Author      string `json:"author"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Thumbnail   string `json:"thumbnail"`
}
