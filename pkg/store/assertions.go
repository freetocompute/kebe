package store

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/asserts"
)

func (s *Store) getSnapRevisionAssertion(c *gin.Context) {
	sha3384digest := c.Param("sha3384digest")
	logrus.Tracef("Requested snap-revision: %s", sha3384digest)

	assertion, err := s.handler.GetSnapRevisionAssertion(sha3384digest, s.rootStoreKey, s.assertsDatabase)
	if err == nil && assertion != nil {
		encodedAssertion := asserts.Encode(assertion)
		logrus.Trace("Sending snap-revision assertion: ")
		logrus.Trace(string(encodedAssertion))

		c.Writer.Header().Set("Content-Type", asserts.MediaType)
		c.Writer.WriteHeader(200)
		_, err = c.Writer.Write(asserts.Encode(assertion))
		if err != nil {
			logrus.Error(err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		return
	} else if err != nil {
		logrus.Error(err)
	} else {
		logrus.Error("unknown error encountered in getSnapRevisionAssertion")
	}

	c.AbortWithStatus(http.StatusBadRequest)
}

func (s *Store) getSnapDeclarationAssertion(c *gin.Context) {
	snapId := c.Param("snap-id")
	logrus.Tracef("Requested snap-declaration: %s", snapId)

	assertion, err := s.handler.GetSnapDeclarationAssertion(snapId, s.rootStoreKey, s.assertsDatabase)
	if err == nil && assertion != nil {
		encodedAssertion := asserts.Encode(assertion)
		logrus.Trace("Sending snap-declaraction assertion: ")
		logrus.Trace(string(encodedAssertion))

		c.Writer.Header().Set("Content-Type", asserts.MediaType)

		_, err = c.Writer.Write(asserts.Encode(assertion))
		if err == nil {
			c.Writer.WriteHeader(200)
			return
		}

		logrus.Error(err)
	} else if err != nil {
		logrus.Error(err)
	} else {
		logrus.Error("unknown error encountered in getSnapDeclarationAssertion")
	}

	c.AbortWithStatus(http.StatusBadRequest)
}

func (s *Store) getAccountAssertion(c *gin.Context) {
	id := c.Param("id")
	logrus.Tracef("Requested account: %s", id)

	accountAssertion, err := s.handler.GetAccountAssertion(id, s.rootStoreKey, s.signingDB)
	if err == nil && accountAssertion != nil {
		assertionBytes := asserts.Encode(accountAssertion)
		c.Writer.Header().Set("Content-Type", asserts.MediaType)
		c.Writer.WriteHeader(200)

		_, err2 := c.Writer.Write(assertionBytes)
		if err2 != nil {
			logrus.Error(err2)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		return
	} else if err != nil {
		logrus.Error(err)
	} else {
		logrus.Errorf("Unknown error encountered trying to get account assertion for account id=%s", id)
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Store) getAccountKey(c *gin.Context) {
	key := c.Param("key")
	logrus.Tracef("Requested account-key: %s", key)

	accountKeyAssertion, err := s.handler.GetAccountKeyAssertion(key, s.rootStoreKey, s.signingDB)
	if err == nil && accountKeyAssertion != nil {
		logrus.Tracef("Found account-key assertion: %+v", accountKeyAssertion)

		c.Writer.WriteHeader(200)
		assertionBytes := asserts.Encode(accountKeyAssertion)
		_, err = c.Writer.Write(assertionBytes)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		return
	} else if err != nil {
		logrus.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	logrus.Error("Unknown error encountered while trying to get account key")
	c.AbortWithStatus(http.StatusInternalServerError)
}
