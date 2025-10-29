package handlers

import (
	"context"
	"time"

	"hh-vacancy-bot/internal/bot/utils"
	"hh-vacancy-bot/internal/models"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

// /start command
func HandleStart(ctx *Context) tele.HandlerFunc {
	return func(c tele.Context) error {
		userID := c.Sender().ID
		userName := c.Sender().Username
		firstName := c.Sender().FirstName
		lastName := c.Sender().LastName

		ctx.Logger.Info("user started bot",
			zap.Int64("user_id", userID),
			zap.String("username", userName),
		)

		dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// get
		user, err := ctx.Store.GetUser(dbCtx, userID)
		if err != nil {
			ctx.Logger.Error("get user failed", zap.Int64("user_id", userID), zap.Error(err))
			return c.Send("😔 Ошибка. Попробуйте позже.")
		}

		if user == nil {
			user = &models.User{
				ID:             userID,
				Username:       stringPtr(userName),
				FirstName:      stringPtr(firstName),
				LastName:       stringPtr(lastName),
				CheckEnabled:   false, // default OFF
				NotifyInterval: 60,    // 1h
			}
			if err := ctx.Store.CreateUser(dbCtx, user); err != nil {
				ctx.Logger.Error("failed to create user", zap.Int64("user_id", userID), zap.Error(err))
				return c.Send("😔 Ошибка при регистрации. Попробуйте позже.")
			}
			ctx.Logger.Info("new user created", zap.Int64("user_id", userID))
		} else {
			needUpdate := false
			if (user.Username == nil && userName != "") || (user.Username != nil && *user.Username != userName) {
				user.Username = stringPtr(userName)
				needUpdate = true
			}
			if (user.FirstName == nil && firstName != "") || (user.FirstName != nil && *user.FirstName != firstName) {
				user.FirstName = stringPtr(firstName)
				needUpdate = true
			}
			if (user.LastName == nil && lastName != "") || (user.LastName != nil && *user.LastName != lastName) {
				user.LastName = stringPtr(lastName)
				needUpdate = true
			}
			if needUpdate {
				if err := ctx.Store.UpdateUser(dbCtx, user); err != nil {
					ctx.Logger.Warn("failed to update user meta", zap.Int64("user_id", userID), zap.Error(err))
				}
			}
			ctx.Logger.Debug("existing user", zap.Int64("user_id", userID))
		}

		// welcome
		name := firstName
		if name == "" {
			name = "пользователь"
		}
		welcomeMsg := utils.FormatWelcomeMessage(name)

		return c.Send(
			welcomeMsg,
			utils.MainMenuKeyboard(),
			tele.ModeMarkdownV2,
		)
	}
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}