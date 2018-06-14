package commands

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/golangci/golangci-lint/pkg/config"
	"github.com/golangci/golangci-lint/pkg/printers"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func setupLog(isVerbose bool) {
	log.SetFlags(0) // don't print time
	if isVerbose {
		logrus.SetLevel(logrus.InfoLevel)
	}
}

func (e *Executor) persistentPreRun(cmd *cobra.Command, args []string) {
	if e.cfg.Run.PrintVersion {
		fmt.Fprintf(printers.StdOut, "golangci-lint has version %s built from %s on %s\n", e.version, e.commit, e.date)
		os.Exit(0)
	}

	runtime.GOMAXPROCS(e.cfg.Run.Concurrency)

	setupLog(e.cfg.Run.IsVerbose)

	if e.cfg.Run.CPUProfilePath != "" {
		f, err := os.Create(e.cfg.Run.CPUProfilePath)
		if err != nil {
			logrus.Fatal(err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			logrus.Fatal(err)
		}
	}
}

func (e *Executor) persistentPostRun(cmd *cobra.Command, args []string) {
	if e.cfg.Run.CPUProfilePath != "" {
		pprof.StopCPUProfile()
	}
	if e.cfg.Run.MemProfilePath != "" {
		f, err := os.Create(e.cfg.Run.MemProfilePath)
		if err != nil {
			logrus.Fatal(err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			logrus.Fatal("could not write memory profile: ", err)
		}
	}

	os.Exit(e.exitCode)
}

func getDefaultConcurrency() int {
	if os.Getenv("HELP_RUN") == "1" {
		return 8 // to make stable concurrency for README help generating builds
	}

	return runtime.NumCPU()
}

func (e *Executor) initRoot() {
	rootCmd := &cobra.Command{
		Use:   "golangci-lint",
		Short: "golangci-lint is a smart linters runner.",
		Long:  `Smart, fast linters runner. Run it in cloud for every GitHub pull request on https://golangci.com`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				logrus.Fatal(err)
			}
		},
		PersistentPreRun:  e.persistentPreRun,
		PersistentPostRun: e.persistentPostRun,
	}

	initRootFlagSet(rootCmd.PersistentFlags(), e.cfg, e.needVersionOption())
	e.rootCmd = rootCmd
}

func initRootFlagSet(fs *pflag.FlagSet, cfg *config.Config, needVersionOption bool) {
	fs.BoolVarP(&cfg.Run.IsVerbose, "verbose", "v", false, wh("verbose output"))
	fs.StringVar(&cfg.Run.CPUProfilePath, "cpu-profile-path", "", wh("Path to CPU profile output file"))
	fs.StringVar(&cfg.Run.MemProfilePath, "mem-profile-path", "", wh("Path to memory profile output file"))
	fs.IntVarP(&cfg.Run.Concurrency, "concurrency", "j", getDefaultConcurrency(), wh("Concurrency (default NumCPU)"))
	if needVersionOption {
		fs.BoolVar(&cfg.Run.PrintVersion, "version", false, wh("Print version"))
	}
}
