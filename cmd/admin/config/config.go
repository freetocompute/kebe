package config

import "github.com/freetocompute/kebe/cmd/admin/config/configkey"

var DefaultValues = map[string]interface{}{
	configkey.KebeAPIURL:  "http://localhost:30080",
	configkey.StoreAPIURL: "http://localhost:30080",
}
