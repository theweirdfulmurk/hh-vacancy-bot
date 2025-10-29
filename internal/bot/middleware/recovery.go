package middleware

import (
	"fmt"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

// Recovery middleware for panic handling
func Recovery(logger *zap.Logger) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			defer func() {
				if r := recover(); r != nil {
					// log panic
					logger.Error("panic recovered",
						zap.Any("panic", r),
						zap.Stack("stack"),
						zap.Int64("user_id", c.Sender().ID),
					)

					// send error msg to user
					_ = c.Send("😔 Произошла ошибка. Пожалуйста, попробуйте позже.")
				}
			}()

			return next(c)
		}
	}
}

func SafeReply(c tele.Context, message string, opts ...interface{}) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic in reply: %v\n", r)
		}
	}()

	return c.Reply(message, opts...)
}

func SafeSend(c tele.Context, message string, opts ...interface{}) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic in send: %v\n", r)
		}
	}()

	return c.Send(message, opts...)
}