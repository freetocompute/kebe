package models

import "gorm.io/gorm"

type Key struct {
	gorm.Model
	Name             string
	SHA3384          string `gorm:"unique"`
	EncodedPublicKey string
	AccountID        uint
	Account          Account
}

type SSHKey struct {
	gorm.Model
	PublicKeyString string `gorm:"unique"`
	AccountID        uint
	Account          Account
}

type Account struct {
	gorm.Model
	// AccountId is the same as publisher-id and developer-id
	AccountId   string `gorm:"unique"`
	DisplayName string `gorm:"unique"`
	Username    string `gorm:"unique"`
	Keys        []Key
	SnapEntries []SnapEntry
	SSHKeys []SSHKey
	Email 			 string
}
