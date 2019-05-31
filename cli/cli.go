/*
 Copyright 2019 Padduck, LLC
  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at
  	http://www.apache.org/licenses/LICENSE-2.0
  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
*/

package cli

import (
	"github.com/pufferpanel/apufferi/logging"
	"github.com/pufferpanel/pufferd/cli/commands"
	"github.com/pufferpanel/pufferd/config"
	"github.com/pufferpanel/pufferd/version"
	"github.com/spf13/cobra"
	"os"
	"runtime"
)

var rootCmd = &cobra.Command{
	Use:   "pufferd",
	Short: "pufferpanel daemon",
}

var configPath string
var loggingLevel string

func init() {
	cobra.OnInitialize(load)

	rootCmd.AddCommand(
		commands.LicenseCmd,
		commands.ShutdownCmd,
		commands.RunCmd,
		commands.ReloadCmd)

	path := "config.json"
	if runtime.GOOS == "linux" {
		path = "/etc/pufferd/config.json"
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", path, "Path to the config to use")
	rootCmd.PersistentFlags().StringVar(&loggingLevel, "logging", "INFO", "Logging level to print to stdout")
	rootCmd.SetVersionTemplate(version.Display)
}

func Execute() error {
	return rootCmd.Execute()
}

func load() {
	config.SetPath(configPath)
	_ = config.LoadConfig()

	level := logging.GetLevel(loggingLevel)
	if level == nil {
		level = logging.INFO
	}

	logging.SetLevel(os.Stdout, level)

	var logPath = config.Get().Data.LogFolder
	_ = logging.WithLogDirectory(logPath, logging.DEBUG, nil)
}
