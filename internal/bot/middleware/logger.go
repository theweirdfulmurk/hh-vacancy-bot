package middleware

import (
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

// Logger middleware for logging all incoming msgs
func Logger(logger *zap.Logger) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			start := time.Now()

			// get user info
			user := c.Sender()
			var userID int64
			var username string

			if user != nil {
				userID = user.ID
				username = user.Username
			}

			// get msg info
			message := c.Message()
			var messageText string
			var messageType string

			if message != nil {
				messageText = message.Text
				messageType = "message"
			}

			callback := c.Callback()
			if callback != nil {
				messageText = callback.Data
				messageType = "callback"
			}

			err := next(c)

			duration := time.Since(start)

			fields := []zap.Field{
				zap.Int64("user_id", userID),
				zap.String("username", username),
				zap.String("type", messageType),
				zap.String("text", messageText),
				zap.Duration("duration", duration),
			}

			if err != nil {
				fields = append(fields, zap.Error(err))
				logger.Error("handler error", fields...)
			} else {
				logger.Info("request handled", fields...)
			}

			return err
		}
	}
}