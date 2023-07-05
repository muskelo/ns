package main

import (
	"github.com/caarlos0/env/v8"

	"github.com/muskelo/ns_server/storage/internal/filemanager"
	"github.com/muskelo/ns_server/storage/internal/server"
)

type config struct {
	FileManagerRoot string `env:"NS_STORAGE_FM_ROOT" envDefault:"/var/ns/default"`
	Listen          string `env:"NS_STORAGE_LISTEN" envDefault:"0.0.0.0:5200"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		panic(err)
	}

	fm := &filemanager.FileManager{
		Root: cfg.FileManagerRoot,
	}
	s := server.New(fm)
	err := server.Serve(cfg.Listen, s)
	if err != nil {
		panic(err)
	}
}
