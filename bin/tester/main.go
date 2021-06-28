package main

import (
	"fmt"
	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/pkg/auth"
	"gopkg.in/macaroon.v2"
	"strings"
)

func MustNew(rootKey, id []byte, loc string, vers macaroon.Version) *macaroon.Macaroon {
	m, err := macaroon.New(rootKey, id, loc, vers)
	if err != nil {
		panic(err)
	}
	return m
}

func getRootDischarge(tokensString string) (string, string) {
	var root string
	var discharge string
	tokens := strings.Split(tokensString, ",")
	for _, t := range tokens {
		fmt.Println(t)
		if strings.Contains(t, "root=") {
			root = strings.TrimPrefix(t, "root=")
		} else {
			discharge = strings.TrimPrefix(t, " discharge=")
		}
	}

	return root, discharge
}

func main() {

	tokenString := "root=MDAxOGxvY2F0aW9uIGEgbG9jYXRpb24KMDAxN2lkZW50aWZpZXIgc29tZSBpZAowMDIyY2lkIGlzLWF1dGhvcml6ZWQtb3Itd2hhdGV2ZXIKMDA1MXZpZCBP1qUdAO8u3ZDcMjW8jJA2EXO4CTWTXO-NTw--2rNxgyeaEY1XZqnKs8qHxi05zxKjNlLrHGkfXmaNaeKulvTNHb65GPtRnJQKMDAxZWNsIGxvZ2luLmJhc2UxMjcubmV0Ojg4OTAKMDA4N2NpZCB7InBlcm1pc3Npb25zIjogWyJwYWNrYWdlX2FjY2VzcyIsICJwYWNrYWdlX21hbmFnZSIsICJwYWNrYWdlX3B1c2giLCAicGFja2FnZV9yZWdpc3RlciIsICJwYWNrYWdlX3JlbGVhc2UiLCAicGFja2FnZV91cGRhdGUiXX0KMDAyZnNpZ25hdHVyZSD3TRk_HxYu_f8CvKXcI_pIQ0SlCwkbbuCuALLYF5mBNwo, discharge=AgEPcmVtb3RlIGxvY2F0aW9uAhlpcy1hdXRob3JpemVkLW9yLXdoYXRldmVyAAAGIMQEaButrqcZPvpIUGSwevMw2VZcj27NAy6lFwa0u3Y6"
	root, discharge := getRootDischarge(tokenString)

	rootM, _ := auth.MacaroonDeserialize(root)
	dischargeM, _ := auth.MacaroonDeserialize(discharge)

	fmt.Printf("%+v\n", rootM)
	fmt.Printf("%+v\n", dischargeM)

	err := rootM.Verify([]byte(config.Secret1), func(caveat string) error {
		return nil
	}, []*macaroon.Macaroon{dischargeM})

	if err != nil {
		panic(err)
	}

	fmt.Println("OMG THIS WORKED?!?!?!")
}
