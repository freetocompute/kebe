package auth

import (
	"encoding/base64"
	"fmt"
	macaroonv2 "gopkg.in/macaroon.v2"
)

// Generate - Generate a Macaroon from the Token configuration
func Generate(secret string, t *Token) (*macaroonv2.Macaroon, error){
	// create a new macaroon
	m, err := macaroonv2.New([]byte(secret), []byte(t.ID), t.Location, macaroonv2.Version(t.Version))
	if err != nil {
		return nil, fmt.Errorf("error creating new macaroon: %w", err)
	}

	for _, caveat := range t.ThirdPartyCaveats {
		for _, c := range caveat {
			bs := []byte(c.Id)
			err := m.AddThirdPartyCaveat([]byte(secret),
				bs, c.Location)
			if err != nil {
				return nil, fmt.Errorf("failed to add caveat: %w", err)
			}
		}
	}

	return m, nil
}

type Caveat struct {
	Id string
	Location string
}

// Caveats - configuration representation of key pair caveats
type Caveats []Caveat

// Token - configuration representation of token metadata attributes
type Token struct {
	ID                string
	Version           int
	Location          string
	ThirdPartyCaveats []Caveats
}

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