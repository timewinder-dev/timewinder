package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/gookit/color"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/timewinder-dev/timewinder/cas"
	"github.com/timewinder-dev/timewinder/model"
)

var (
	debugFlag       bool
	keepGoing       bool
	detailsFlag     bool
	noDeadlocksFlag bool
	terminationFlag bool
	maxDepth        int
	cpuProfile      string
	memProfile      string
	lruSize         int
	parallelFlag    bool
	execThreads     int
	checkThreads    int
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
	runCmd.Flags().BoolVar(&terminationFlag, "termination", false, "Require all threads to terminate (reject infinite loops)")
	runCmd.Flags().IntVar(&maxDepth, "max-depth", 0, "Maximum depth to explore (0 = unlimited)")
	runCmd.Flags().StringVar(&cpuProfile, "cpuprofile", "", "Write CPU profile to specified file")
	runCmd.Flags().StringVar(&memProfile, "memprofile", "", "Write memory profile to specified file")
	runCmd.Flags().IntVar(&lruSize, "lru-size", 10000, "LRU cache size for CAS (0 = disabled)")
	runCmd.Flags().BoolVar(&parallelFlag, "parallel", false, "Use multi-threaded model checker for better performance")
	runCmd.Flags().IntVar(&execThreads, "exec-threads", 0, "Number of execution worker threads (0 = NumCPU, only with --parallel)")
	runCmd.Flags().IntVar(&checkThreads, "check-threads", 0, "Number of checking worker threads (0 = NumCPU/2, only with --parallel)")
}

func runCommand(cmd *cobra.Command, args []string) {
	// Set up CPU profiling if requested
	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			log.Fatal().Err(err).Msg("Could not create CPU profile file")
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal().Err(err).Msg("Could not start CPU profile")
		}
		defer pprof.StopCPUProfile()
	}

	// Set up memory profiling if requested (written at end of function)
	if memProfile != "" {
		defer func() {
			f, err := os.Create(memProfile)
			if err != nil {
				log.Fatal().Err(err).Msg("Could not create memory profile file")
			}
			defer f.Close()
			runtime.GC() // Get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatal().Err(err).Msg("Could not write memory profile")
			}
		}()
	}

	filename := args[0]
	spec, err := model.LoadSpecFromFile(filename)
	if err != nil {
		log.Fatal().Err(err).Msg("Couldn't load specfile")
	}

	// Create CAS with optional LRU caching
	memoryCAS := cas.NewMemoryCAS()
	var casStore cas.CAS
	if lruSize == 0 {
		// LRU disabled
		casStore = memoryCAS
	} else {
		// LRU enabled with specified size
		casStore = cas.NewLRUCache(memoryCAS, lruSize)
	}

	exec, err := spec.BuildExecutor(casStore)
	if err != nil {
		log.Fatal().Err(err).Msg("Couldn't build executor for specfile")
	}
	if debugFlag {
		exec.DebugWriter = os.Stderr
	} else {
		exec.DebugWriter = io.Discard
	}

	// Set up progress reporter (always enabled for user visibility)
	exec.Reporter = &model.ColorReporter{Writer: os.Stderr}

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
	// CLI flags override spec settings
	if noDeadlocksFlag {
		exec.NoDeadlocks = true
	}
	if terminationFlag {
		exec.Termination = true
	}
	if maxDepth > 0 {
		exec.MaxDepth = maxDepth
	}
	// Set parallel execution flags
	if parallelFlag {
		exec.UseMultiThread = true
		exec.NumExecThreads = execThreads
		exec.NumCheckThreads = checkThreads
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

	// Check if result matches expectations
	matchesExpectation := spec.MatchesExpectedResult(result)

	// Print success/failure message based on expectations
	if matchesExpectation {
		fmt.Fprintln(os.Stderr)
		if spec.Spec.ExpectedError != "" {
			fmt.Fprintln(os.Stderr, color.Green.Sprintf("✓ Model checking completed as expected - found expected error: %s", spec.Spec.ExpectedError))
		} else {
			fmt.Fprintln(os.Stderr, color.Green.Sprint("✓ Model checking completed successfully - all properties satisfied!"))
		}
	} else {
		fmt.Fprintln(os.Stderr)
		if spec.Spec.ExpectedError != "" {
			fmt.Fprintln(os.Stderr, color.Red.Sprintf("✗ Expected error '%s' but got different result", spec.Spec.ExpectedError))
			os.Exit(1)
		}
		// If no expected error, violations are already printed above
	}
}
