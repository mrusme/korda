package view

import (
	"encoding/json"

	"tty.fail/mrus/korda/helpers/jsontime"
	logicnotifications "tty.fail/mrus/korda/logic/root/notifications"
	"tty.fail/mrus/korda/models/notification"
)

type Notification struct {
	ID          string             `json:"id"`
	CreatedAt   jsontime.Time      `json:"createdAt"`
	Level       notification.Level `json:"level"`
	Type        notification.Type  `json:"type"`
	Title       string             `json:"title"`
	Description *string            `json:"description,omitempty"`
	Data        json.RawMessage    `json:"data,omitempty"`
	ReadAt      *jsontime.Time     `json:"readAt,omitempty"`
}

type NotificationTemplate struct {
	Name string `json:"name"`
	HTML string `json:"html"`
}

func NewNotification(n *notification.Notification) Notification {
	out := Notification{
		ID:        id(n.ID),
		CreatedAt: time3(n.CreatedAt),
		Level:     n.Level,
		Type:      n.Type,
		Title:     n.Title,
		ReadAt:    nullTime(n.ReadAt),
	}
	if n.Description.Valid {
		out.Description = &n.Description.V
	}
	if n.Data.Valid {
		out.Data = json.RawMessage(n.Data.V)
	}

	return out
}

func NewNotifications(list []notification.Notification) []Notification {
	out := make([]Notification, 0, len(list))
	for i := range list {
		out = append(out, NewNotification(&list[i]))
	}

	return out
}

func NewNotificationTemplate(t logicnotifications.Template) NotificationTemplate {
	return NotificationTemplate{Name: t.Name, HTML: t.HTML}
}
