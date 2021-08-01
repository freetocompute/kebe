package store

import (
	"crypto/rsa"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/freetocompute/kebe/pkg/store/responses"

	"github.com/freetocompute/kebe/pkg/store/requests"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/asserts"
	"github.com/snapcore/snapd/asserts/assertstest"
)

type Store struct {
	assertsDatabase   *asserts.Database
	rootStoreKey      *rsa.PrivateKey
	genericPrivateKey *rsa.PrivateKey
	signingDB         *assertstest.SigningDB
	handler           IStoreHandler
}

func New(handler IStoreHandler, assertsDB *asserts.Database, rootStoreKey *rsa.PrivateKey, genericPrivateKey *rsa.PrivateKey, signingDB *assertstest.SigningDB) *Store {
	return &Store{
		// db:                db,
		assertsDatabase:   assertsDB,
		rootStoreKey:      rootStoreKey,
		signingDB:         signingDB,
		genericPrivateKey: genericPrivateKey,
		handler:           handler,
	}
}

func (s *Store) snapDownload(c *gin.Context) {
	snapFilename := c.Param("filename")

	bytes, err := s.handler.SnapDownload(snapFilename)
	if err == nil && bytes != nil {
		_, err2 := c.Writer.Write(*bytes)
		if err2 != nil {
			logrus.Error(err2)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		return
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Store) snapRefresh(c *gin.Context) {
	request := c.Request
	writer := c.Writer

	var actionRequest requests.SnapActionRequest
	err := json.NewDecoder(request.Body).Decode(&actionRequest)
	if err != nil {
		logrus.Error(err)
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	writer.Header().Set("Content-Type", "application/json")

	snapActionResultList, err := s.handler.SnapRefresh(&actionRequest.Actions)
	if err == nil && snapActionResultList != nil {
		c.JSON(http.StatusOK, &snapActionResultList)
		return
	} else if err != nil {
		logrus.Error(err)
	} else {
		logrus.Error("unknown error encountered in snapRefresh")
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Store) getSnapSections(c *gin.Context) {
	writer := c.Writer
	logrus.Trace("/api/v1/snaps/sections")
	writer.Header().Set("Content-Type", "application/hal+json")

	result, err := s.handler.GetSections()
	if err == nil && result != nil {
		c.JSON(http.StatusOK, result)
	} else if err != nil {
		logrus.Error(err)
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Store) findSnap(c *gin.Context) {
	// TODO: implement query parameters
	// q : search term, assume name right now
	name := c.Query("q")
	searchResults, err := s.handler.FindSnap(name)
	if err == nil && searchResults != nil {
		logrus.Infof("%+v", searchResults)

		c.Writer.Header().Set("Content-Type", "application/json")
		bytes, _ := json.Marshal(&searchResults)
		_, err2 := c.Writer.Write(bytes)
		if err2 != nil {
			logrus.Error(err2)
			c.AbortWithStatus(http.StatusInternalServerError)
		}

		return
	} else if err != nil {
		logrus.Error(err)
	} else {
		logrus.Error("unknown error encountered handling /v2/snaps/find in findSnap")
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Store) getSnapNames(c *gin.Context) {
	writer := c.Writer
	logrus.Trace("/api/v1/snaps/names")

	writer.Header().Set("Content-Type", "application/hal+json")

	catalogItems, err := s.handler.GetSnapNames()
	if err == nil && catalogItems != nil {
		bytes, err := json.Marshal(catalogItems)
		if err == nil {
			_, err2 := writer.Write(bytes)
			if err2 == nil {
				return
			}
		}
	}

	if err != nil {
		logrus.Error(err)
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Store) unscannedUpload(c *gin.Context) {
	snapFileData, err := c.FormFile("binary")

	// TODO: fix the actual error response to be something expected
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "No snap file is received",
		})
		return
	}

	file, err := snapFileData.Open()
	defer func(file multipart.File) {
		err2 := file.Close()
		if err2 != nil {
			logrus.Error(err2)
		}
	}(file)
	if err == nil {
		id, err2 := s.handler.UnscannedUpload(file)
		if err2 == nil && id != "" {
			c.JSON(http.StatusOK, &responses.Unscanned{UploadId: id})
		}
	}
}

func (s *Store) authRequestIdPOST(c *gin.Context) {
	resp := s.handler.AuthRequest()
	c.JSON(http.StatusOK, resp)
}

func (s *Store) authDevicePOST(c *gin.Context) {
	request := c.Request
	dec := asserts.NewDecoder(request.Body)
	for {
		got, err := dec.Decode()
		if err == io.EOF {
			break
		}
		if err != nil { // assume broken i/o
			panic(err)
		}
		if got.Type() == asserts.SerialRequestType {
			serialRequest := got.(*asserts.SerialRequest)

			serialAssertion, err2 := s.handler.AuthDevice(serialRequest, asserts.RSAPrivateKey(s.genericPrivateKey), s.signingDB)
			if err2 == nil && serialAssertion != nil {
				encodedSerialAssertion := asserts.Encode(serialAssertion)
				logrus.Trace("Sending serial assertion: ")

				c.Writer.Header().Set("Content-Type", asserts.MediaType)
				c.Writer.WriteHeader(200)
				_, err3 := c.Writer.Write(encodedSerialAssertion)
				if err3 == nil {
					return
				}

				logrus.Error(err3)
			}
		} else {
			logrus.Warningf("Assertion type included but not exepected: %s", got.Type().Name)
		}
	}

	c.AbortWithStatus(http.StatusInternalServerError)
}

func (s *Store) authNonce(c *gin.Context) {
	// TODO: do we need to store this?
	nonce := s.handler.AuthNonce()
	c.JSON(http.StatusOK, &nonce)
}

func (s *Store) authSession(c *gin.Context) {
	// TODO: implement actual sessions?
	session := s.handler.AuthSession()
	c.JSON(http.StatusOK, session)
}
