package models

// BaseEntity represents the base fields for all entities
type BaseEntity struct {
	ID int `gorm:"primaryKey"`
}
