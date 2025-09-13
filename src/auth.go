package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var roturValidationKey = "warpdrive-blogger"

var (
	allowedUsers    []string
	allowedUsersAll bool = false
)

func init() {
	allowedUsers = []string{"mist", "jax"}
}

func validateRoturValidator(validator string) (bool, string) {
	v := strings.TrimSpace(validator)
	if v == "" || roturValidationKey == "" {
		return false, ""
	}
	username := v
	if i := strings.Index(v, ","); i >= 0 {
		username = v[:i]
	}
	if username == "" {
		return false, ""
	}

	endpoint := "https://social.rotur.dev/validate"
	q := url.Values{}
	q.Set("key", roturValidationKey)
	q.Set("v", v)
	req, err := http.NewRequest("GET", endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return false, ""
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, ""
	}
	defer resp.Body.Close()
	var body struct {
		Valid bool `json:"valid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false, ""
	}
	if !body.Valid {
		return false, ""
	}
	return true, username
}

func requireValidator(c *gin.Context) {
	validator := strings.TrimSpace(c.GetHeader("X-Rotur-Validator"))
	if validator == "" {
		auth := c.GetHeader("Authorization")
		const prefix = "Validator "
		if strings.HasPrefix(auth, prefix) {
			validator = strings.TrimSpace(auth[len(prefix):])
		}
	}
	if ok, username := validateRoturValidator(validator); ok {
		c.Set("username", username)
		c.Next()
		return
	}
	c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Error: "invalid or missing validator"})
}

func requireAllowedUser(c *gin.Context) {
	username, ok := validatedUser(c)
	if !ok || username == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Error: "missing validator"})
		return
	}
	if allowedUsersAll {
		c.Next()
		return
	}
	u := strings.ToLower(username)
	for _, a := range allowedUsers {
		if u == a {
			c.Next()
			return
		}
	}
	c.AbortWithStatusJSON(http.StatusForbidden, errorResponse{Error: "user not allowed"})
}

func validatedUser(c *gin.Context) (string, bool) {
	v, ok := c.Get("username")
	if !ok {
		return "", false
	}
	s, _ := v.(string)
	if s == "" {
		return "", false
	}
	return s, true
}
