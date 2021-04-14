package asserts

import (
	"errors"
	"fmt"
	"github.com/freetocompute/kebe/pkg/models"
	"github.com/snapcore/snapd/asserts"
	"time"
)

func MakeSnapDeclarationAssertion(authorityId, publisherId string, snapEntry *models.SnapEntry, storePrivateKey asserts.PrivateKey, db *asserts.Database) (*asserts.SnapDeclaration, error) {
	headers := map[string]interface{}{
		"authority-id": authorityId,
		"series":       "16",
		"snap-id":      snapEntry.SnapStoreID,
		"publisher-id": publisherId,
		"snap-name":    snapEntry.Name,
		"timestamp":    time.Now().Format(time.RFC3339),
		"revision":     "1",
	}

	a, err := db.Sign(asserts.SnapDeclarationType, headers, nil, storePrivateKey.PublicKey().ID())
	if err != nil {
		return nil, err
	}

	if aaa, ok := a.(*asserts.SnapDeclaration); ok {
		return aaa, nil
	}

	return nil, errors.New("unable to cast assertion")
}

func MakeSnapRevisionAssertion(authorityId, digest, snapID string, size uint64, revision int, developerID, keyID string, db *asserts.Database) (*asserts.SnapRevision, error) {
	headers := map[string]interface{}{
		"authority-id":  authorityId,
		"snap-sha3-384": digest,
		"snap-id":       snapID,
		"snap-size":     fmt.Sprintf("%d", size),
		"snap-revision": fmt.Sprintf("%d", revision),
		"developer-id":  developerID,
		"timestamp":     time.Now().Format(time.RFC3339),
	}
	a, err := db.Sign(asserts.SnapRevisionType, headers, nil, keyID)
	if err != nil {
		return nil, err
	}

	aaa, ok := a.(*asserts.SnapRevision)
	if ok {
		return aaa, nil
	}

	return nil, errors.New("unable to cast assertion")
}
