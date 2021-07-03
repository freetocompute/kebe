package store

import (
	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/database"
	"github.com/freetocompute/kebe/pkg/models"
	asserts2 "github.com/freetocompute/kebe/pkg/store/asserts"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/asserts"
	"github.com/snapcore/snapd/asserts/assertstest"
	"gorm.io/gorm/clause"
	"net/http"
)

func (s *Store) getSnapRevisionAssertion(c *gin.Context) {
	sha3384digest := c.Param("sha3384digest")
	logrus.Tracef("Requested snap-revision: %s", sha3384digest)
	//
	var snapRevision models.SnapRevision
	db := s.db.Where("sha3_384", sha3384digest).Find(&snapRevision)
	if db.Error != nil {
		logrus.Error(db.Error)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	} else if db.RowsAffected == 0 {
		c.AbortWithStatus(http.StatusNotFound)
	}

	var snapEntry models.SnapEntry
	db = s.db.Preload(clause.Associations).Where("id", snapRevision.SnapEntryID).Find(&snapEntry)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		writer := c.Writer
		storeAuthorityId := config.MustGetString(configkey.RootAuthority)
		assertion, err := asserts2.MakeSnapRevisionAssertion(storeAuthorityId, sha3384digest, snapEntry.SnapStoreID, uint64(snapRevision.Size), int(snapRevision.ID), snapEntry.Account.AccountId,
			asserts.RSAPrivateKey(s.rootStoreKey).PublicKey().ID(), s.assertsDatabase)

		if err != nil {
			logrus.Error(err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		encodedAssertion := asserts.Encode(assertion)
		logrus.Trace("Sending snap-revision assertion: ")
		logrus.Trace(string(encodedAssertion))

		writer.Header().Set("Content-Type", asserts.MediaType)
		writer.WriteHeader(200)
		writer.Write(asserts.Encode(assertion))
		return
	}

	logrus.Error("no snap entry found")
	c.AbortWithStatus(http.StatusBadRequest)
}

func (s *Store) getSnapDeclarationAssertion(c *gin.Context) {
	snapId := c.Param("snap-id")
	logrus.Tracef("Requested snap-declaration: %s", snapId)

	var snapEntry models.SnapEntry
	db := s.db.Where("snap_store_id", snapId).Preload("Account").Find(&snapEntry)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		writer := c.Writer

		rootAuthorityId := config.MustGetString(configkey.RootAuthority)
		aaa, err := asserts2.MakeSnapDeclarationAssertion(rootAuthorityId, snapEntry.Account.AccountId, &snapEntry, asserts.RSAPrivateKey(s.rootStoreKey), s.assertsDatabase)
		if err != nil {
			logrus.Error(err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		encodedAssertion := asserts.Encode(aaa)
		logrus.Trace("Sending snap-declaraction assertion: ")
		logrus.Trace(string(encodedAssertion))

		writer.Header().Set("Content-Type", asserts.MediaType)
		writer.WriteHeader(200)
		writer.Write(asserts.Encode(aaa))
		return
	}

	logrus.Error("no snap entry found")
	c.AbortWithStatus(http.StatusBadRequest)
}

func (s *Store) getAccountAssertion(c *gin.Context) {
	writer := c.Writer

	id := c.Param("id")
	logrus.Tracef("Requested account: %s", id)

	var account models.Account
	db := s.db.Where("account_id", id).Find(&account)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {

		pk := asserts.RSAPrivateKey(s.rootStoreKey)
		_, bytes := createAccountAssertion(s.signingDB, pk.PublicKey().ID(), account.AccountId, account.Username)
		writer.Header().Set("Content-Type", asserts.MediaType)
		writer.WriteHeader(200)

		logrus.Trace(string(bytes))
		writer.Write(bytes)
		return
	}
}

func (s *Store) getAccountKey(c *gin.Context) {
	writer := c.Writer

	key := c.Param("key")
	logrus.Tracef("Requested account-key: %s", key)

	var accountKey models.Key
	db := s.db.Preload("Account").Where("sha3384", key).Find(&accountKey)
	if db.Error != nil {
		logrus.Fatal(db.Error)
		writer.WriteHeader(500)
	} else {
		if db.RowsAffected == 1 {
			logrus.Infof("Found account-key: %+v", accountKey)

			pbk, err := asserts.DecodePublicKey([]byte(accountKey.EncodedPublicKey))
			if err != nil {
				panic(err)
			}

			trustedAcct := s.getTrustedAccount(accountKey.Account.AccountId, s.signingDB, accountKey.Account.DisplayName)

			trustedAcctKeyHeaders := map[string]interface{}{
				"since":               "2015-11-20T15:04:00Z",
				"until":               "2500-11-20T15:04:00Z",
				"public-key-sha3-384": accountKey.SHA3384,
				"name":                accountKey.Name,
			}
			//
			trustedAccKey := assertstest.NewAccountKey(s.signingDB, trustedAcct, trustedAcctKeyHeaders, pbk, "")
			writer.Header().Set("Content-Type", asserts.MediaType)
			writer.WriteHeader(200)
			assertionBytes := asserts.Encode(trustedAccKey)
			logrus.Trace(string(assertionBytes))
			writer.Write(assertionBytes)
			return
		}
	}

	logrus.Errorf("Unable to find account-key for %s", key)
	writer.WriteHeader(500)
}

func (s *Store) getTrustedAccount(accountID string, signingDB *assertstest.SigningDB, displayName string) *asserts.Account {
	trustedAcctHeaders := map[string]interface{}{
		"validation": "verified",
		"timestamp":  "2015-11-20T15:04:00Z",
	}

	if displayName != "" {
		trustedAcctHeaders["display-name"] = displayName
	}

	trustedAcctHeaders["account-id"] = accountID
	trustedAcct := assertstest.NewAccount(signingDB, accountID, trustedAcctHeaders, "")

	return trustedAcct
}

func createAccountAssertion(signingDB *assertstest.SigningDB, keyId string, accountId string, storeAccountUsername string) (*asserts.Account, []byte) {
	trustedAcctHeaders := map[string]interface{}{
		"validation": "certified",
		"timestamp":  "2015-11-20T15:04:00Z",
		"account-id": accountId,
	}

	trustedAcct := assertstest.NewAccount(signingDB, storeAccountUsername, trustedAcctHeaders, keyId)

	bytes := asserts.Encode(trustedAcct)

	return trustedAcct, bytes
}
