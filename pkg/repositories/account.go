package repositories

import (
	"github.com/freetocompute/kebe/pkg/database"
	"github.com/freetocompute/kebe/pkg/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IAccountRepository interface {
	GetAccountByEmail(email string, preload bool) (*models.Account, error)
	GetAccountById(accountId string, preload bool) (*models.Account, error)
	AddKey(name string, SHA3384 string, encodedPublicKey string, accountEmail string) (*models.Key, error)
	GetKeyBySHA3384(sha3384 string) (*models.Key, error)
}

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (a *AccountRepository) GetKeyBySHA3384(sha3384 string) (*models.Key, error) {
	whereModel := &models.Key{SHA3384: sha3384}
	return a.getKeyByWhereModel(whereModel, true)
}

func (a *AccountRepository) GetAccountById(accountId string, preload bool) (*models.Account, error) {
	whereModel := &models.Account{AccountId: accountId}
	return a.getAccountByWhereModel(whereModel, preload)
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

func (a *AccountRepository) getKeyByWhereModel(whereModel *models.Key, preload bool) (*models.Key, error) {
	var accountKey models.Key
	var db *gorm.DB
	if preload {
		db = a.db.Where(whereModel).Preload(clause.Associations).Find(&accountKey)
	} else {
		db = a.db.Where(whereModel).Find(&accountKey)
	}

	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		return &accountKey, nil
	} else if db.Error != nil {
		return nil, db.Error
	}

	logrus.Warningf("Account key not found: %+v", whereModel)
	return nil, nil
}

func (a *AccountRepository) getAccountByWhereModel(whereModel *models.Account, preload bool) (*models.Account, error) {
	var userAccount models.Account
	var db *gorm.DB
	if preload {
		db = a.db.Where(whereModel).Preload(clause.Associations).Find(&userAccount)
	} else {
		db = a.db.Where(whereModel).Find(&userAccount)
	}

	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		return &userAccount, nil
	} else if db.Error != nil {
		return nil, db.Error
	}

	logrus.Warningf("Account not found: %+v", whereModel)
	return nil, nil
}
