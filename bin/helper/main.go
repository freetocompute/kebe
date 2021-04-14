package main

import (
	"fmt"
	"github.com/freetocompute/kebe/cmd/helper"
	"github.com/freetocompute/kebe/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	cobra.OnInitialize(config.LoadConfig)
}

func main() {
	logrus.SetLevel(logrus.TraceLevel)

	if err := helper.Helper.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
