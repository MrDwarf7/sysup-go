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
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	cfgFile       string
	skip          []string
	continueOnErr bool
	noCache       bool
	doShutdown    bool
	forceShutdown bool
	logLevel      string
)

var rootCmd = &cobra.Command{
	Use:   "sysup-go",
	Short: "System update orchestrator",
	Long: `sysup-go runs an ordered list of user-defined programs from
$XDG_CONFIG_HOME/sysup-go (programs.toml or programs/*.toml).

-s / --skip drops programs by name or alias.
-c / --continue accumulates program errors instead of stopping (unused until later).
These flags are not the same: -c is continue, -s c skips the program aliased c.`,
}

// Execute runs the root command and maps errors to process exit codes.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(exitCode(err))
	}
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.StringVar(&cfgFile, "config", "", "config file (default is $XDG_CONFIG_HOME/sysup-go/config.toml)")
	pf.StringSliceVarP(&skip, "skip", "s", nil, "skip programs by name or alias")
	pf.BoolVarP(&continueOnErr, "continue", "c", false, "continue after program errors")
	pf.BoolVar(&noCache, "no-cache", false, "skip end-of-run cache sweep")
	pf.BoolVarP(&doShutdown, "shutdown", "d", false, "shut down after a clean run")
	pf.BoolVar(&forceShutdown, "force-shutdown", false, "shut down even if programs failed")
	pf.StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
}

func newLogger() *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(logLevel) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
}
