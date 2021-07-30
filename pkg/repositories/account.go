package repositories

import (
	"github.com/freetocompute/kebe/pkg/database"
	"github.com/freetocompute/kebe/pkg/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IAccountRepository interface {
	GetAccountByEmail(email string, preload bool) (*models.Account, error)
	AddKey(name string, SHA3384 string, encodedPublicKey string, accountEmail string) (*models.Key, error)
}

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (a *AccountRepository) GetAccountByEmail(email string, preload bool) (*models.Account, error) {
	var userAccount models.Account
	var db *gorm.DB
	if preload {
		db = a.db.Where(&models.Account{Email: email}).Preload(clause.Associations).Find(&userAccount)
	} else {
		db = a.db.Where(&models.Account{Email: email}).Find(&userAccount)
	}

	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		return &userAccount, nil
	}

	return nil, db.Error
}

func (a *AccountRepository) AddKey(name string, SHA3384 string, encodedPublicKey string, email string) (*models.Key, error) {
	acct, err := a.GetAccountByEmail(email, false)
	if err == nil && acct != nil {
		accountKeyToAdd := models.Key{
			Name:             name,
			SHA3384:          SHA3384,
			AccountID:        acct.ID,
			EncodedPublicKey: encodedPublicKey,
		}

		a.db.Save(&accountKeyToAdd)
		return &accountKeyToAdd, nil
	}

	return nil, err
}
