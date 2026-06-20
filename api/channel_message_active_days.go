package api

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	"sealchat/model"
	"sealchat/service"
)

type channelMessageActiveDaysResponse struct {
	ChannelID string   `json:"channel_id"`
	Month     string   `json:"month"`
	Timezone  string   `json:"timezone"`
	Days      []string `json:"days"`
}

func ChannelMessageActiveDaysHandler(c *fiber.Ctx) error {
	user := c.Locals("user").(*model.UserModel)
	channelID := strings.TrimSpace(c.Params("channelId"))
	if channelID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "channel_id 不能为空"})
	}
	if err := validateExportChannel(user.ID, channelID); err != nil {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{"message": err.Error()})
	}

	month := strings.TrimSpace(c.Query("month"))
	days, err := service.ListChannelMessageActiveDays(channelID, month)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(channelMessageActiveDaysResponse{
		ChannelID: channelID,
		Month:     month,
		Timezone:  "local",
		Days:      days,
	})
}
