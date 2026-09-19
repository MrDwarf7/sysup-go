package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"sysup-go/internal/config"
	"sysup-go/internal/logfmt"
	"sysup-go/internal/program"
)

// app is one CLI invocation. Flags, the file-backed Config, and the
// filesystem live here instead of package globals filled by init().
type app struct {
	fs     afero.Fs
	cfg    config.Config
	origin string

	skip           []string
	continueOnErr  bool
	logLevel       slog.Level
	logFile        string
	logFileOut     *os.File
	generateConfig bool
}

func Execute() {
	if err := NewRoot().Execute(); err != nil {
		os.Exit(ExitCode(err))
	}
}

// NewRoot builds the command tree. Call this instead of relying on init().
func NewRoot() *cobra.Command {
	a := &app{}
	root := &cobra.Command{
		Use:   config.AppName,
		Short: "System update orchestrator",
		Long: `Runs user-defined programs from ` + filepath.Join("$XDG_CONFIG_HOME", config.AppName) +
			` (` + program.FileName + ` or ` + program.DirName + `/*.toml).

-s / --skip drops programs by name or alias.
Skipping a name also drops later programs named name-*
(so -s m skips mirror and mirror-stage / mirror-backup / mirror-swap).
-c / --continue accumulates program errors instead of stopping.`,
		PersistentPreRunE: a.setup,
		RunE:              a.runPlan,
	}
	a.bindFlags(root)
	cobra.CheckErr(viper.BindPFlags(root.PersistentFlags()))
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return &FlagError{Err: err}
	})
	root.AddCommand(a.listCmd(), a.validateCmd())
	return root
}

func (a *app) bindFlags(root *cobra.Command) {
	pf := root.PersistentFlags()
	configHelp := "Config file (default " +
		filepath.Join("$XDG_CONFIG_HOME", config.AppName, config.ConfigFileName) + ")"

	pf.String("config", "", configHelp)
	pf.Bool("generate-config", false, "Write default "+config.ConfigFileName+" and exit")
	pf.StringSliceP("skip", "s", nil, "Skip programs by name or alias (also drops name-* children)")
	pf.BoolP("continue", "c", false, "Continue after program errors")
	pf.Bool("no-cache", false, "Skip end-of-run cache sweep")
	pf.BoolP("shutdown", "d", false, "Shut down after a clean run")
	pf.Bool("force-shutdown", false, "Shut down even if programs failed")
	pf.String("log-level", "info", "Log level (debug, info, warn, error)")
	pf.String("log-file", logfmt.DefaultFile(), "Also write logs to this file (empty or - to disable)")
}

func (a *app) configPath() (string, error) {
	if p := viper.GetString("config"); p != "" {
		return p, nil
	}
	return config.AppConfig()
}

// skipSetup is true for cobra's own machinery. Those commands must not
// load or generate config: stdout is the completion script / choice list
// (see cobra "Generating shell completions").
//
// Names that fire this:
//   - help
//   - completion and its children (bash, zsh, fish, powershell)
//   - __complete / __completeNoDesc (hidden; the shell calls these on tab)
func skipSetup(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "help", "completion", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
			return true
		}
	}
	return false
}

func (a *app) setup(cmd *cobra.Command, _ []string) error {
	// Flag parse errors happen before PersistentPreRun, so they still
	// print usage. RunE failures (step exit 1, nothing to run) should not.
	cmd.SilenceUsage = true
	if skipSetup(cmd) {
		return nil
	}

	a.fs = afero.NewOsFs()
	viper.SetFs(a.fs)

	// Env last: viper.find is override > flag > env > file > default.
	// AutomaticEnv is a lookup switch, not a snapshot; it must be on
	// before the first Get. Replacer makes log-level -> LOG_LEVEL.
	viper.SetEnvPrefix(config.AppName)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	a.skip = viper.GetStringSlice("skip")
	a.continueOnErr = viper.GetBool("continue")
	a.logFile = viper.GetString("log-file")
	a.generateConfig = viper.GetBool("generate-config")
	if err := a.logLevel.UnmarshalText([]byte(viper.GetString("log-level"))); err != nil {
		return fmt.Errorf("invalid log-level %q", viper.GetString("log-level"))
	}

	path, err := a.configPath()
	if err != nil {
		return err
	}

	if a.generateConfig {
		return a.writeDefaultsAndExit(cmd, path)
	}

	if viper.GetString("config") == "" {
		empty, emptyErr := config.MissingOrEmpty(a.fs, path)
		if emptyErr != nil {
			return emptyErr
		}
		if empty {
			return a.writeDefaultsAndExit(cmd, path)
		}
	}

	v := config.NewViper(a.fs, path)
	a.cfg, err = config.Load(v)
	if err != nil {
		return err
	}
	a.origin = v.ConfigFileUsed()
	if a.origin == "" {
		a.origin = "default"
	}
	return nil
}

func (a *app) writeDefaultsAndExit(cmd *cobra.Command, path string) error {
	if err := config.Generate(a.fs, path); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "wrote default config to %s\n", path); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}

func (a *app) logger(stderr *os.File) *slog.Logger {
	if stderr == nil {
		stderr = os.Stderr
	}
	console := logfmt.New(stderr, logfmt.Options{Level: a.logLevel, Color: logfmt.ColorTTY(stderr)})
	if logfmt.Disabled(a.logFile) {
		return slog.New(console)
	}
	f, err := os.OpenFile(a.logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		fmt.Fprintf(stderr, "log-file %s: %v\n", a.logFile, err)
		return slog.New(console)
	}
	a.logFileOut = f
	file := logfmt.New(f, logfmt.Options{Level: a.logLevel, Color: false})
	return slog.New(slog.NewMultiHandler(console, file))
}
