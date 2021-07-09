package assertions

import (
	"github.com/snapcore/snapd/asserts"
	"io/ioutil"
)

func EncodeAssertionToFile(assertionPath string, assertion asserts.Assertion) {
	bytes := asserts.Encode(assertion)
	err := ioutil.WriteFile(assertionPath, bytes, 0644)
	if err != nil {
		panic(err)
	}
}

func GetPublicKeyFromBody(bodyBytes []byte) (asserts.PublicKey, error){
	pubkey, err := asserts.DecodePublicKey(bodyBytes)
	if err != nil {
		return nil, err
	}

	return pubkey, nil
}