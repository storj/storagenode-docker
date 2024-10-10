package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"storj.io/storagenode-docker/supervisor"
	"storj.io/storj/private/version/checker"
)

type config struct {
	Interval             time.Duration
	CheckTimeout         time.Duration
	BinaryLocation       string
	VersionServerAddress string
}

func main() {
	ctx := getContext()

	rootCmd := &cobra.Command{
		Use:   "supervisor",
		Short: "A process manager for the storagenode",
	}

	var cfg config
	runCmd := &cobra.Command{
		Use:     "run [flags] -- [command]",
		Short:   "Run the supervisor",
		Example: `supervisor run --check-interval=10 -- /path/to/storagenode run --config-dir=/path/to/config`,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSupervisor(ctx, cfg, args)
		},
	}

	runCmd.Flags().DurationVarP(&cfg.Interval, "check-interval", "", 15*time.Minute, "Interval in seconds to check for updates")
	runCmd.Flags().DurationVarP(&cfg.CheckTimeout, "check-timeout", "", 1*time.Minute, "Request timeout for checking for updates")
	runCmd.Flags().StringVarP(&cfg.BinaryLocation, "binary-location", "", "/app/bin/storagenode", "Location of the binary to run")
	runCmd.Flags().StringVarP(&cfg.VersionServerAddress, "version-server-address", "", "https://version.storj.io", "URL of the version server")
	rootCmd.AddCommand(runCmd)

	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		log.Fatalln(err)
	}
}

func runSupervisor(ctx context.Context, cfg config, args []string) error {
	process := supervisor.NewProcess(cfg.BinaryLocation, args)

	versionChecker := checker.New(checker.ClientConfig{
		ServerAddress:  cfg.VersionServerAddress,
		RequestTimeout: cfg.CheckTimeout,
	})

	updater := supervisor.NewUpdater(cfg.VersionServerAddress, cfg.Interval)

	return nil
}

func getContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Printf("Got a signal from the OS: %q", sig)
		signal.Stop(c)
		cancel()
	}()

	return ctx
}
