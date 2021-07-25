package auth

import (
	"encoding/base64"
	macaroonv2 "gopkg.in/macaroon.v2"
)

// MacaroonSerialize returns a store-compatible serialized representation of the given macaroon
func MacaroonSerialize(m *macaroonv2.Macaroon) (string, error) {
	marshalled, err := m.MarshalBinary()
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(marshalled)
	return encoded, nil
}

// MacaroonDeserialize returns a deserialized macaroon from a given store-compatible serialization
func MacaroonDeserialize(serializedMacaroon string) (*macaroonv2.Macaroon, error) {
	var m macaroonv2.Macaroon
	decoded, err := base64.RawURLEncoding.DecodeString(serializedMacaroon)
	if err != nil {
		return nil, err
	}
	err = m.UnmarshalBinary(decoded)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func MustNewMacaroon(rootKey, id []byte, loc string, vers macaroonv2.Version) *macaroonv2.Macaroon {
	m, err := macaroonv2.New(rootKey, id, loc, vers)
	if err != nil {
		panic(err)
	}
	return m
}
