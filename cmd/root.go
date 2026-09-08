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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"sysup-go/internal/config"
	"sysup-go/internal/program"
)

var cfgFile string

var (
	skip          []string
	continueOnErr bool
	noCache       bool
	doShutdown    bool
	forceShutdown bool
	logLevel      string
)

var rootCmd = &cobra.Command{
	Use:   config.AppName,
	Short: "System update orchestrator",
	Long: `Runs user-defined programs from ` + filepath.Join("$XDG_CONFIG_HOME", config.AppName) +
		` (` + program.FileName + ` or ` + program.DirName + `/*.toml).

-s / --skip drops programs by name or alias.
-c / --continue accumulates program errors instead of stopping.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(ExitCode(err))
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	pf := rootCmd.PersistentFlags()

	// "Config file (default is $XDG_CONFIG_HOME/"+config.AppName+"/"+config.ConfigFileName+")"

	sb := strings.Builder{}
	sb.WriteString("Config file (default ")
	sb.WriteString(
		filepath.Join("$XDG_CONFIG_HOME", config.AppName, config.ConfigFileName),
	)
	sb.WriteString(")")

	pf.StringVar(&cfgFile, "config", "", sb.String())
	pf.StringSliceVarP(&skip, "skip", "s", nil, "Skip programs by name or alias")
	pf.BoolVarP(&continueOnErr, "continue", "c", false, "Continue after program errors")
	pf.BoolVar(&noCache, "no-cache", false, "Skip end-of-run cache sweep")
	pf.BoolVarP(&doShutdown, "shutdown", "d", false, "Shut down after a clean run")
	pf.BoolVar(&forceShutdown, "force-shutdown", false, "Shut down even if programs failed")
	pf.StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	cobra.CheckErr(viper.BindPFlags(pf))
}

func initConfig() {
	fsys := afero.NewOsFs()
	viper.SetFs(fsys)

	if cfgFile != "" {
		info, err := fsys.Stat(cfgFile)
		cobra.CheckErr(err)
		if info.IsDir() {
			cobra.CheckErr(fmt.Errorf("config path is a directory: %s", cfgFile))
		}
		viper.SetConfigFile(cfgFile)
	} else {
		dir, err := config.AppDir(fsys)
		cobra.CheckErr(err)
		viper.AddConfigPath(dir)
		viper.SetConfigType(config.ConfigType)
		viper.SetConfigName(config.ConfigName)
	}

	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		fileViper := viper.New()
		fileViper.SetFs(fsys)
		fileViper.SetConfigFile(viper.ConfigFileUsed())
		fileViper.SetConfigType(config.ConfigType)
		cobra.CheckErr(fileViper.ReadInConfig())
		_, strictErr := config.UnmarshalStrict(fileViper)
		cobra.CheckErr(strictErr)
		return
	}
	if cfgFile != "" {
		cobra.CheckErr(err)
		return
	}
	if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok {
		return
	}
	cobra.CheckErr(err)
}

func newLogger() *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(viper.GetString("log-level"))); err != nil {
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
}
