package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/profclems/go-dotenv"
	"github.com/spf13/cobra"
	"github.com/zeebo/errs"

	"storj.io/common/identity"
	"storj.io/common/storj"
	"storj.io/common/sync2"
	"storj.io/common/version"
	"storj.io/storagenode-docker/supervisor"
	"storj.io/storj/private/version/checker"
)

type config struct {
	Interval             time.Duration `env:"STORJ_SUPERVISOR_UPDATE_CHECK_INTERVAL" default:"15m" description:"Interval in seconds to check for updates"`
	CheckTimeout         time.Duration `env:"STORJ_SUPERVISOR_UPDATE_CHECK_TIMEOUT" default:"1m" description:"Request timeout for checking for updates"`
	BinaryLocation       string        `env:"STORJ_SUPERVISOR_BINARY_LOCATION" default:"/app/bin/storagenode" description:"Path to the storagenode binary"`
	BinaryStoreDir       string        `env:"STORJ_SUPERVISOR_BINARY_STORE_DIR" default:"/app/config/bin" description:"Directory to store storagenode backup binaries"`
	VersionServerAddress string        `env:"STORJ_SUPERVISOR_VERSION_SERVER_ADDRESS" default:"https://version.storj.io" description:"URL of the version server"`

	NodeID      storj.NodeID `env:"STORJ_SUPERVISOR_NODE_ID" description:"Node ID. If not provided, it will be read from the identity file"`
	IdentityDir string       `env:"STORJ_SUPERVISOR_IDENTITY_DIR" default:"/app/identity" description:"Path to the identity directory. Required if node ID is not provided"`
}

func main() {
	ctx := getContext()
	slog.SetDefault(slog.With("service", "supervisor"))

	rootCmd := &cobra.Command{
		Use:   "supervisor",
		Short: "A process manager for the storagenode",
	}

	var cfg config
	execCmd := &cobra.Command{
		Use:     "exec STORAGENODE_COMMAND",
		Short:   "Execute the storagenode binary with supervisor",
		Example: `supervisor exec /path/to/storagenode run --config-dir=/path/to/config`,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env := dotenv.New()
			err := env.Unmarshal(&cfg)
			if err != nil {
				return err
			}

			log.Println(cfg)
			return execSupervisor(ctx, cfg, args)
		},
		DisableFlagParsing: true,
	}

	rootCmd.AddCommand(execCmd)

	err := rootCmd.ExecuteContext(ctx)
	if err != nil && !errs.Is(err, context.Canceled) {
		slog.Info("error executing command", "error", err)
		os.Exit(1)
	}
}

func execSupervisor(ctx context.Context, cfg config, args []string) error {
	if cfg.NodeID.IsZero() {
		fullID, err := identity.Config{
			CertPath: filepath.Join(cfg.IdentityDir, "identity.cert"),
			KeyPath:  filepath.Join(cfg.IdentityDir, "identity.key"),
		}.Load()

		if err != nil {
			return err
		}
		cfg.NodeID = fullID.ID
	}

	process := supervisor.NewProcess(cfg.NodeID, cfg.BinaryLocation, cfg.BinaryStoreDir, args)

	versionChecker := checker.New(checker.ClientConfig{
		ServerAddress:  cfg.VersionServerAddress,
		RequestTimeout: cfg.CheckTimeout,
	})

	updater := supervisor.NewUpdater(versionChecker)

	// check that storagenode binary exists
	if _, err := os.Stat(cfg.BinaryLocation); err != nil {
		// check store dir for backup binary
		backupBinary := filepath.Join(cfg.BinaryStoreDir, "storagenode")
		if _, err := os.Stat(backupBinary); err == nil {
			// copy backup binary to binary location
			if err := copyBinary(ctx, cfg.BinaryLocation, backupBinary); err != nil {
				return err
			}
		} else {
			log.Println("Binary does not exist, downloading new binary")
			// binary does not exist, download it
			_, err := updater.Update(ctx, process, version.SemVer{})
			if err != nil {
				return err
			}
		}
	}

	err := supervisor.New(updater, process, cfg.Interval).Run(ctx)
	if err != nil {
		slog.Info("Supervisor stopped", "error", err)
		return err
	}

	return nil
}

func getContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-c
		slog.Info("Got a signal from the OS:", "signal", sig)
		signal.Stop(c)
		cancel()
	}()

	return ctx
}

func copyBinary(ctx context.Context, dest, src string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return errs.Wrap(err)
	}
	defer func() {
		err = errs.Combine(err, srcFile.Close())
	}()

	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return errs.Wrap(err)
	}

	defer func() {
		err = errs.Combine(err, destFile.Close())
	}()

	_, err = sync2.Copy(ctx, destFile, srcFile)
	if err != nil {
		return errs.Wrap(err)
	}

	return nil
}
