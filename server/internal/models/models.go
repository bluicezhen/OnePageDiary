package models

import "time"

type Entry struct {
	Date      string    `gorm:"primaryKey;size:10"`
	Content   string    `gorm:"type:text"`
	Revision  uint64
	Deleted   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SyncEvent struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	Date      string    `gorm:"size:10;index"`
	Action    string    `gorm:"size:10"`
	Revision  uint64
	CreatedAt time.Time
}

type User struct {
	ID           uint `gorm:"primaryKey;autoIncrement"`
	Username     string `gorm:"uniqueIndex;size:64"`
	PasswordHash string `gorm:"size:255"`
	CreatedAt    time.Time
}
