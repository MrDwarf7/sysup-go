/*
Copyright © 2026 MrDwarf7 // Blake B.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
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
	origin config.Origin

	cfgFile        string
	skip           []string
	continueOnErr  bool
	noCache        bool
	doShutdown     bool
	forceShutdown  bool
	logLevel       string
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
	root.AddCommand(a.listCmd(), a.validateCmd())
	return root
}

func (a *app) bindFlags(root *cobra.Command) {
	pf := root.PersistentFlags()
	configHelp := "Config file (default " +
		filepath.Join("$XDG_CONFIG_HOME", config.AppName, config.ConfigFileName) + ")"

	pf.StringVar(&a.cfgFile, "config", "", configHelp)
	pf.BoolVar(&a.generateConfig, "generate-config", false, "Write default "+config.ConfigFileName+" and exit")
	pf.StringSliceVarP(&a.skip, "skip", "s", nil, "Skip programs by name or alias (also drops name-* children)")
	pf.BoolVarP(&a.continueOnErr, "continue", "c", false, "Continue after program errors")
	pf.BoolVar(&a.noCache, "no-cache", false, "Skip end-of-run cache sweep")
	pf.BoolVarP(&a.doShutdown, "shutdown", "d", false, "Shut down after a clean run")
	pf.BoolVar(&a.forceShutdown, "force-shutdown", false, "Shut down even if programs failed")
	pf.StringVar(&a.logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	pf.StringVar(&a.logFile, "log-file", logfmt.DefaultFile(), "Also write logs to this file (empty or - to disable)")
}

func (a *app) configPath() (string, error) {
	if a.cfgFile != "" {
		return a.cfgFile, nil
	}
	return config.AppConfig(a.fs)
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

	path, err := a.configPath()
	if err != nil {
		return err
	}

	if a.generateConfig {
		return a.writeDefaultsAndExit(cmd, path)
	}

	if a.cfgFile == "" {
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
	a.origin = config.OriginFrom(v)

	// Env last: viper.find is override > flag > env > file > default.
	// AutomaticEnv is a lookup switch, not a snapshot; it must be on
	// before the first Get. Replacer makes log-level -> LOG_LEVEL.
	viper.SetEnvPrefix(config.AppName)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()
	a.skip = viper.GetStringSlice("skip")
	a.logLevel = viper.GetString("log-level")
	a.logFile = viper.GetString("log-file")
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
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(a.logLevel)); err != nil {
		lvl = slog.LevelInfo
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	console := logfmt.New(stderr, logfmt.Options{Level: lvl, Color: logfmt.ColorTTY(stderr)})
	if logfmt.Disabled(a.logFile) {
		return slog.New(console)
	}
	f, err := os.OpenFile(a.logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		fmt.Fprintf(stderr, "log-file %s: %v\n", a.logFile, err)
		return slog.New(console)
	}
	a.logFileOut = f
	file := logfmt.New(f, logfmt.Options{Level: lvl, Color: false})
	return slog.New(slog.NewMultiHandler(console, file))
}
