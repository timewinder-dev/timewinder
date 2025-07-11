package main

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/timewinder-dev/timewinder/model"
)

var runCmd = &cobra.Command{
	Use:   "run SPECFILE",
	Short: "Run the model",
	Args:  cobra.MinimumNArgs(1),
	Run:   runCommand,
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
	err = exec.Initialize()
	if err != nil {
		log.Fatal().Err(err).Msg("Couldn't init executor engine")
	}
	b, err := json.Marshal(*exec.InitialState)
	if err != nil {
		log.Fatal().Err(err).Msg("can't serialize")
		return
	}
	exec.Program.DebugPrint()
	fmt.Println(string(b))
	//err = exec.RunModel()
	//if err != nil {
	//log.Fatal().Err(err).Msg("Error running model")
	//}
}
