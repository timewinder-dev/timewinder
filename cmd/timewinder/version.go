package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of timewinder",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("timewinder version 1.0.0")
	},
}