package admin

import (
	"fmt"
	"github.com/freetocompute/kebe/cmd/store"
	"github.com/freetocompute/kebe/config"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"sort"
)

func init() {
	Admin.AddCommand(store.Store)
	Admin.AddCommand(info)
	Admin.AddCommand(login)
	Admin.AddCommand(account)
}

var Admin = &cobra.Command{
	Use: "kebe-admin",
	TraverseChildren: true,
}

var info = &cobra.Command{
	Use:   "info",
	Short: "info",
	Run: func(cmd *cobra.Command, args []string) {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name", "Value"})

		var defaultValueKeys []string
		for k, _ := range config.DefaultValues {
			defaultValueKeys = append(defaultValueKeys, k)
		}

		sort.Strings(defaultValueKeys)

		logrus.Infof("Defaults were: ")
		for _, k := range defaultValueKeys {
			table.Append([]string{k, fmt.Sprintf("%+v", config.DefaultValues[k])})
		}
		table.Render()

		table = tablewriter.NewWriter(os.Stdout)
		logrus.Infof("Config values")
		table.SetHeader([]string{"Name", "Value"})

		logrus.Infof("Actual values: ")

		allKeys := viper.AllKeys()
		sort.Strings(allKeys)

		for _, k := range allKeys {
			table.Append([]string{k, fmt.Sprintf("%+v", viper.Get(k))})
		}
		table.Render()
	},
}
