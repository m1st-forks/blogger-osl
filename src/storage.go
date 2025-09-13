package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

var postsFilePath = "posts.json"
var postsDir = "posts"
var thumbsDir = "thumbnails"

type storeFile struct {
	NextID int64  `json:"next_id"`
	Posts  []Post `json:"posts"`
}

func saveToDiskLocked() error {
	dir := filepath.Dir(postsFilePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	ps := make([]Post, 0, len(posts))
	for _, p := range posts {
		ps = append(ps, *p)
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].ID < ps[j].ID })

	sf := storeFile{NextID: nextID, Posts: ps}
	b, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return err
	}
	tmp := postsFilePath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, postsFilePath)
}

func loadFromDisk() error {
	postsMu.Lock()
	defer postsMu.Unlock()

	b, err := os.ReadFile(postsFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var sf storeFile
	if err := json.Unmarshal(b, &sf); err != nil {
		return err
	}
	posts = map[int64]*Post{}
	nextID = 1
	if sf.NextID > 1 {
		nextID = sf.NextID
	}
	for i := range sf.Posts {
		p := sf.Posts[i]
		posts[p.ID] = &p
		if p.ID >= nextID {
			nextID = p.ID + 1
		}
	}
	return nil
}

func contentFilePath(id int64) string {
	return filepath.Join(postsDir, fmt.Sprintf("%d.md", id))
}

func writeContentLocked(id int64, content string) error {
	dir := postsDir
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	path := contentFilePath(id)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readContent(id int64) (string, error) {
	b, err := os.ReadFile(contentFilePath(id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(b), nil
}

func deleteContentLocked(id int64) error {
	err := os.Remove(contentFilePath(id))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
