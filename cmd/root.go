package cmd

import (
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configDir string

	RootCmd = &cobra.Command{
		Use:   "user.app",
		Short: "Root command for the user.app application",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			initConfig()
		},
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.AddCommand(applicationCmd)
}

func initConfig() {
	viper.AddConfigPath(configDir)
	viper.SetConfigName("config")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal().Err(err).Msg("unable to read config.")
	}
}

func init() {
	configDefaultDir, err := os.Getwd()
	if err != nil {
		log.Error().Err(err).Msg("cannot get the working directory")
	}

	// Viper Initialize
	viper.SetConfigType("toml")

	RootCmd.PersistentFlags().StringVar(&configDir, "config", filepath.Join(configDefaultDir, "configs"), "Config file location")
}
