package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"database/sql"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	_ "github.com/sijms/go-ora/v2"
	"github.com/tijanadmi/ugo_evid/cmd/api"
	"github.com/tijanadmi/ugo_evid/docs" // Swagger generated files
	db "github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/util"
	"time"
)

var interruptSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGINT,
}

// @securityDefinitions.apikey bearerAuth
// @in header
// @name Authorization
// @TokenUrl /users/login  // dodato

func main() {
	// Swagger 2.0 Meta Information
	docs.SwaggerInfo.Title = "TDI - Evidence"
	docs.SwaggerInfo.Description = "TDI - Backend for Evidence app"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "localhost:8080"
	//docs.SwaggerInfo.BasePath = "/api/v1"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config")
	}

	if config.Environment == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	ctx, stop := signal.NotifyContext(context.Background(), interruptSignals...)
	defer stop()

	connPool, err := sql.Open("oracle", config.DBSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to db")
	}

	defer connPool.Close()
	connPool.SetMaxOpenConns(20)
	connPool.SetMaxIdleConns(5)
	connPool.SetConnMaxLifetime(30 * time.Minute)
	pingCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := connPool.PingContext(pingCtx); err != nil {
		log.Fatal().Err(err).Msg("cannot ping Oracle")
	}
	store := db.NewStore(connPool)
	fmt.Println("Connected to Oracle successfully!")
	runGinServer(config, store)

}

func runGinServer(config util.Config, store db.Store) {
	server, err := api.NewServer(config, store)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create server")
	}

	err = server.Start(config.HTTPServerAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot start server")
	}
}
