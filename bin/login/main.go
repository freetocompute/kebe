package main

import (
	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/pkg/login/server"
	"github.com/spf13/cobra"
)

func init() {
	cobra.OnInitialize(config.LoadConfig)
}

func main() {
	s := &server.Server{}
	s.Run()
}

