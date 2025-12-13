package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/timewinder-dev/timewinder/model"
)

var debugFlag bool

var runCmd = &cobra.Command{
	Use:   "run SPECFILE",
	Short: "Run the model",
	Args:  cobra.MinimumNArgs(1),
	Run:   runCommand,
}

func init() {
	runCmd.Flags().BoolVar(&debugFlag, "debug", false, "Enable debug output to see each execution step")
}

func runCommand(cmd *cobra.Command, args []string) {
	filename := args[0]
	spec, err := model.LoadSpecFromFile(filename)
	if err != nil {
		log.Fatal().Err(err).Msg("Couldn't load specfile")
	}
	exec, err := spec.BuildExecutor()
	if err != nil {
		log.Fatal().Err(err).Msg("Couldn't build executor for specfile")
	}
	if debugFlag {
		exec.DebugWriter = os.Stderr
	} else {
		exec.DebugWriter = io.Discard
	}
	err = exec.Initialize()
	if err != nil {
		log.Fatal().Err(err).Msg("Couldn't init executor engine")
	}
	b, err := json.Marshal(*exec.InitialState)
	if err != nil {
		log.Fatal().Err(err).Msg("can't serialize")
		return
	}
	if debugFlag {
		exec.Program.DebugPrint()
		fmt.Fprintf(os.Stderr, "Initial state: %s\n\n", string(b))
	}

	fmt.Println("Running model checker...")
	err = exec.RunModel()
	if err != nil {
		log.Fatal().Err(err).Msg("Error running model")
	}

	fmt.Println("Model checking completed successfully - all properties satisfied!")
}
