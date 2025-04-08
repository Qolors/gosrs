package core

import "github.com/qolors/gosrs/internal/core/model"

type Client interface {
	GetPlayerData() (model.StampedData, error)
}

type Notifier interface {
	SendNotification(day_data []model.StampedData, session_data []model.StampedData) error
}

type Storage interface {
	GetAll() []model.StampedData
	Add(model.StampedData) bool
}
