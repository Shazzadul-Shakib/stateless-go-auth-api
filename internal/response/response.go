// Package response writes success responses in the same envelope style as apperr's errors.
package response

import "github.com/gin-gonic/gin"

type body struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data"`
}

// JSON writes {"data": data, "message": message}; message is omitted when empty.
func JSON(c *gin.Context, status int, data any, message string) {
	c.JSON(status, body{Success: true, Message: message, Data: data})
}
