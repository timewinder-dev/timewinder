package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	logLevel string
)

var rootCmd = &cobra.Command{
	Use:   "timewinder",
	Short: "A brief description of your application",
	Long:  "A longer description of your application",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Set up zerolog
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

		// Parse and set log level
		level, err := zerolog.ParseLevel(logLevel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid log level '%s', using 'info'\n", logLevel)
			level = zerolog.InfoLevel
		}
		zerolog.SetGlobalLevel(level)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Set log level (trace, debug, info, warn, error)")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(runCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}