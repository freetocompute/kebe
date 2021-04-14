package store

import (
	"github.com/spf13/cobra"
)

func init() {
	Store.AddCommand(&Initialize)
	Store.AddCommand(&Destroy)
	Store.AddCommand(&RegenerateAssertions)
}

var Store = &cobra.Command{
	Use:   "store",
	Long:  "store",
	Short: "store",
}
