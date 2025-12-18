package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/gookit/color"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/timewinder-dev/timewinder/model"
)

var (
	debugFlag      bool
	keepGoing      bool
	detailsFlag    bool
	noDeadlocksFlag bool
)

var runCmd = &cobra.Command{
	Use:   "run SPECFILE",
	Short: "Run the model",
	Args:  cobra.MinimumNArgs(1),
	Run:   runCommand,
}

func init() {
	runCmd.Flags().BoolVar(&debugFlag, "debug", false, "Enable debug output to see each execution step")
	runCmd.Flags().BoolVar(&keepGoing, "keep-going", false, "Keep checking the model after reporting it's first error")
	runCmd.Flags().BoolVar(&detailsFlag, "details", false, "Show detailed trace reconstruction when property violations occur")
	runCmd.Flags().BoolVar(&noDeadlocksFlag, "no-deadlocks", false, "Disable deadlock detection (allow states where no threads can progress)")
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
	if keepGoing {
		exec.KeepGoing = true
	}
	if detailsFlag {
		exec.ShowDetails = true
	}
	if noDeadlocksFlag {
		exec.NoDeadlocks = true
	}

	fmt.Fprintln(os.Stderr, color.Cyan.Sprint("Running model checker..."))

	result, err := exec.RunModel()

	// Handle execution errors (e.g., runtime panic, invalid state)
	if err != nil {
		log.Fatal().Err(err).Msg("Error during model checking")
	}

	// Handle initialization errors (no result available)
	if result == nil {
		log.Fatal().Msg("Model checking failed to produce a result")
	}

	// Print violations if any occurred
	if !result.Success {
		if keepGoing && len(result.Violations) > 0 {
			// Print all violations
			fmt.Fprint(os.Stderr, model.FormatAllViolations(result.Violations))
		} else if len(result.Violations) > 0 {
			// Print first violation
			fmt.Fprint(os.Stderr, model.FormatPropertyViolation(result.Violations[0]))
		}
	}

	// Always print statistics at the bottom
	fmt.Fprint(os.Stderr, model.FormatStatistics(result.Statistics))

	// Print success message if no violations
	if result.Success {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, color.Green.Sprint("✓ Model checking completed successfully - all properties satisfied!"))
	}
}
