package handlers

import (
	"hh-vacancy-bot/internal/bot/utils"

	tele "gopkg.in/telebot.v3"
)

// /help
func HandleHelp(ctx *Context) tele.HandlerFunc {
	return func(c tele.Context) error {
		helpMsg := utils.FormatHelpMessage()

		return c.Send(
			helpMsg,
			utils.MainMenuKeyboard(),
			tele.ModeMarkdownV2,
		)
	}
}