// Package cli implements the omnivore command.
package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
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
	"golang.org/x/term"
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
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(cmd.Flags())
			if err != nil {
				return err
			}
			if err := promptSecrets(&cfg, cmd.ErrOrStderr()); err != nil {
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
	cmd.MarkFlagRequired("command")
	return cmd
}

// promptSecrets asks for the passwords that cfg needs but doesn't have, on the terminal.
func promptSecrets(cfg *config.Config, w io.Writer) error {
	interactive := term.IsTerminal(int(os.Stdin.Fd()))
	read := func(prompt string) (string, error) {
		fmt.Fprint(w, prompt)
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(w)
		return string(b), err
	}
	if cfg.AskPass && cfg.Password == "" {
		if !interactive {
			return fmt.Errorf("--ask-pass needs a terminal; set %s instead", config.PasswordEnv)
		}
		pw, err := read("SSH password: ")
		if err != nil {
			return err
		}
		cfg.Password = pw
	}
	if cfg.Sudo && cfg.SudoPassword == "" && interactive {
		pw, err := read("sudo password (leave empty for passwordless sudo): ")
		if err != nil {
			return err
		}
		cfg.SudoPassword = pw
	}
	return nil
}

// Run runs cfg.Command on cfg.Hosts, saving history under base. The results are shown in the
// interactive UI, or written to out if cfg.NoUI is set; then Run returns an error if any host
// failed.
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
	command := cfg.Command
	if cfg.Sudo {
		command = "sudo " + command
	}
	st := state.New(command, cfg.Hosts)
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
		snap := st.Snapshot()
		if err := ui.PrintSummary(out, snap); err != nil {
			return err
		}
		// Let scripts detect failures from the exit status.
		if failed := saveFailed(hist, snap, out, log); failed > 0 {
			return fmt.Errorf("%d of %d hosts failed", failed, len(snap.Hosts))
		}
		return nil
	}

	if err := ui.Run(ctx, st, logs); err != nil {
		return err
	}
	cancel()
	<-finished
	fmt.Fprintf(out, "Output saved in %s\n", hist.Dir)
	saveFailed(hist, st.Snapshot(), out, log)
	return nil
}

// saveFailed writes the failed hosts to the history, so that they can be retried with
// --hosts-file, and returns how many there were.
func saveFailed(hist *store.Store, snap state.Snapshot, out io.Writer, log *slog.Logger) int {
	failed := snap.Names(func(h state.Host) bool { return h.Status == state.Failed })
	if len(failed) == 0 {
		return 0
	}
	path, err := hist.WriteHostList(store.FailedFile, failed)
	if err != nil {
		log.Warn("can't save failed hosts", "err", err)
		return len(failed)
	}
	fmt.Fprintf(out, "%d failed hosts saved; retry them with: --hosts-file %s\n", len(failed), path)
	return len(failed)
}
