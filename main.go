package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"user.app/cmd"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if err := cmd.RootCmd.Execute(); err != nil {
		log.Fatal().Msg("unable to execute root command.")
	}
}
