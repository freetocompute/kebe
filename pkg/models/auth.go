package models

import "gorm.io/gorm"

type AuthRequest struct {
	gorm.Model
}

type Serial struct {
	gorm.Model
	SerialUUID string
}