package core

import "github.com/qolors/gosrs/internal/core/model"

type Client interface {
	GetPlayerData() (model.StampedData, error)
}

type Notifier interface {
	SendNotification([]model.StampedData) error
}

type Storage interface {
	GetAll() []model.StampedData
	Add(model.StampedData) bool
}
