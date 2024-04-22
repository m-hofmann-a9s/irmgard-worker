package main

import "github.com/rs/zerolog/log"

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatal().Err(err).Msg(msg)
	}
}
