// Package cli implements the omnivore command.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/discoriver/omnivore/internal/config"
	"github.com/discoriver/omnivore/internal/logging"
	"github.com/discoriver/omnivore/internal/runner"
	"github.com/discoriver/omnivore/internal/state"
	"github.com/discoriver/omnivore/internal/store"
	"github.com/discoriver/omnivore/internal/ui"
	"github.com/spf13/cobra"
)

// logLines is the number of log lines kept for the UI's log panel.
const logLines = 1000

// Execute runs the omnivore command with the process's arguments.
func Execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return NewCommand().ExecuteContext(ctx)
}

// NewCommand returns the omnivore command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "omnivore",
		Short: "Omnivore devours all SSH output, and provides intelligent grouping.",
		Long: "Omnivore runs a command on many hosts over SSH and groups hosts with identical output, " +
			"so that anomalies and unexpected output stand out.",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(cmd.Flags())
			if err != nil {
				return err
			}
			base, err := store.DefaultBase()
			if err != nil {
				return err
			}
			return Run(cmd.Context(), cfg, base, cmd.OutOrStdout())
		},
	}
	config.RegisterFlags(cmd.Flags())
	cmd.MarkFlagRequired("hosts")
	cmd.MarkFlagRequired("command")
	return cmd
}

// Run runs cfg.Command on cfg.Hosts, saving history under base. The results are shown in the
// interactive UI, or written to out if cfg.NoUI is set.
func Run(ctx context.Context, cfg config.Config, base string, out io.Writer) error {
	hist, err := store.New(base)
	if err != nil {
		return err
	}
	logFile, err := hist.OpenLog()
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	defer logFile.Close()

	logs := logging.NewBuffer(logLines)
	log := logging.New(logFile, logs)
	log.Info("omnivore started", "hosts", len(cfg.Hosts), "history", hist.Dir, "config", cfg.File)

	mcfg, err := cfg.Massh(log)
	if err != nil {
		return err
	}

	// Stop the remote commands when we return, whether the user quit or everything finished.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	events, err := mcfg.Stream(ctx)
	if err != nil {
		return err
	}
	st := state.New(cfg.Command, cfg.Hosts)
	finished := make(chan struct{})
	go func() {
		runner.Run(events, st, hist, log)
		close(finished)
	}()

	if cfg.NoUI {
		select {
		case <-finished:
		case <-ctx.Done():
			<-finished
			return ctx.Err()
		}
		return ui.PrintSummary(out, st.Snapshot())
	}

	if err := ui.Run(ctx, st, logs); err != nil {
		return err
	}
	cancel()
	<-finished
	fmt.Fprintf(out, "Output saved in %s\n", hist.Dir)
	return nil
}
