package kebe

import (
	"encoding/json"
	"github.com/freetocompute/kebe/pkg/kebe/apiobjects"
	"github.com/freetocompute/kebe/pkg/models"
	"github.com/freetocompute/kebe/pkg/objectstore"
	"github.com/freetocompute/kebe/pkg/sha"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"mime/multipart"
	"net/http"
	"path"
	"path/filepath"
)

type Kebe struct {
	db *gorm.DB
}

func NewKebe(db *gorm.DB) *Kebe {
	return &Kebe{
		db: db,
	}
}

func (k *Kebe) SetupEndpoints(r *gin.Engine) {
	r.POST("/kebe/v1/accounts/add", k.addAccount)
	r.POST("/kebe/v1/snaps/add", k.addSnap)
	r.POST("/kebe/v1/snaps/upload/:id", k.uploadSnap)
	r.POST("/kebe/v1/keys/add", k.addKey)
}

func (k *Kebe) saveFileToTemp(c *gin.Context, snapFile *multipart.FileHeader) string {
	// Retrieve file information
	extension := filepath.Ext(snapFile.Filename)
	// Generate random file name for the new uploaded file so it doesn't override the old file with same name
	newFileName := uuid.New().String() + extension

	// The file is received, so let's save it
	if err := c.SaveUploadedFile(snapFile, "/tmp/"+newFileName); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": "Unable to save the file",
		})
		return ""
	}

	return newFileName
}

func (k *Kebe) uploadSnap(c *gin.Context) {
	snapFile, err := c.FormFile("snap")
	// The file cannot be received.
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "No snap file is received",
		})
		return
	}
	assertionFile, err := c.FormFile("assertion")
	// The file cannot be received.
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "No assertion file is received",
		})
		return
	}

	snapId, found := c.GetPostForm("snapId")
	// The file cannot be received.
	if !found {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "No snap id",
		})
		return
	}

	var snapEntry models.SnapEntry
	res := k.db.Where("snap_store_id", snapId).Find(&snapEntry)
	if res.RowsAffected == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "Snap does not exist",
		})
		return
	}

	snapFileName := k.saveFileToTemp(c, snapFile)
	assertionFileName := k.saveFileToTemp(c, assertionFile)

	objStore := objectstore.NewObjectStore()
	err = objStore.SaveFileToBucket("snaps", path.Join("/", "tmp", snapFileName))
	if err != nil {
		logrus.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
	}
	err = objStore.SaveFileToBucket("assertions", path.Join("/", "tmp", assertionFileName))
	if err != nil {
		logrus.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
	}

	// create revision, add it to the snap entry
	sha3384dgst, size, err := sha.SnapFileSHA3_384(path.Join("/", "tmp", snapFileName))

	revision := models.SnapRevision{
		SnapFilename:           snapFileName,
		BuildAssertionFilename: assertionFileName,
		SnapEntryID:            snapEntry.ID,
		SHA3_384:               sha3384dgst,
		Size:                   int64(size),
	}

	k.db.Save(&revision)

	snapEntry.LatestRevisionID = revision.ID

	k.db.Save(&snapEntry)

	// File saved successfully. Return proper result
	c.JSON(http.StatusOK, gin.H{
		"message": "Your file has been successfully uploaded.",
	})
}

func (k *Kebe) addSnap(c *gin.Context) {
	var snap apiobjects.Snap
	json.NewDecoder(c.Request.Body).Decode(&snap)

	snapID := uuid.New().String()
	logrus.Infof("Adding snap: %s, snap store id: %s", snap.Name, snapID)

	// get developer
	var developer models.Account
	k.db.Where("account_id", snap.DeveloperID).Find(&developer)

	snapModel := &models.SnapEntry{
		Name:        snap.Name,
		SnapStoreID: snapID,
		AccountID:   developer.ID,
		Type:        snap.Type,
	}

	res := k.db.Save(snapModel)

	snap.SnapStoreID = snapModel.SnapStoreID

	if res.Error != nil {
		logrus.Error(res.Error)
		c.AbortWithError(http.StatusInternalServerError, res.Error)
		return
	}

	c.JSON(http.StatusCreated, &snap)
}

func (k *Kebe) addAccount(c *gin.Context) {
	var account apiobjects.Account
	_ = json.NewDecoder(c.Request.Body).Decode(&account)

	logrus.Infof("Adding account display-name: %s, username: %s", account.DisplayName, account.Username)

	res := k.db.Save(&models.Account{
		Username:    account.Username,
		DisplayName: account.DisplayName,
		AccountId:   uuid.New().String(),
	})

	if res.Error != nil {
		logrus.Error(res.Error)
		c.AbortWithError(http.StatusInternalServerError, res.Error)
		return
	}

	c.JSON(http.StatusCreated, &account)
}

func (k *Kebe) addKey(c *gin.Context) {
	var key apiobjects.Key
	_ = json.NewDecoder(c.Request.Body).Decode(&key)

	var account models.Account
	k.db.Where("account_id", key.AccountId).Find(&account)

	k.db.Save(&models.Key{
		Name:             key.Name,
		SHA3384:          key.SHA3384,
		AccountID:        account.ID,
		EncodedPublicKey: key.EncodedPublicKey,
	})
}
