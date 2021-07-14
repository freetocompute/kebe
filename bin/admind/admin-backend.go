package main

import (
	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/pkg/admind"
	"github.com/spf13/cobra"
)

func init() {
	cobra.OnInitialize(config.LoadConfig)
}

func main() {
	s := &admind.Server{}
	s.Init()
	s.Run()
}

