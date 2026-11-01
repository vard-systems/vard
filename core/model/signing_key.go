package model

import "time"

type SigningKey struct {
	KID        string    `gorm:"column:kid;primaryKey;size:64"`
	Algorithm  string    `gorm:"size:16;not null;index"`
	PrivateKey []byte    `gorm:"not null"`
	CreatedAt  time.Time `gorm:"not null"`
	// RetiredAt is set when a newer key takes over; a retired key is still
	// published, so its tokens verify until they expire.
	RetiredAt *time.Time
}

func (SigningKey) TableName() string {
	return "signing_keys"
}
