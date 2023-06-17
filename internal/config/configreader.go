package configreader

import (
	"file-watcher/internal/app/structs"
	"fmt"
	"github.com/spf13/viper"
	"log"
	"net/http"
)

func ProcessConfig(cflag string) *structs.Config {
	var config structs.Config

	if cflag == "" {
		viper.SetConfigName("application.json")
	} else {
		viper.SetConfigName(cflag)
	}
	viper.SetConfigType("json")
	viper.AddConfigPath("./conf")

	// Читаем файл конфигурации
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(fmt.Errorf("ошибка чтения конфигурации: %w", err))
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		log.Fatal("Error parsing configuration")
	}

	if config.Debug {
		go func() { // http://localhost:6060/debug/pprof/
			e := http.ListenAndServe("localhost:6060", nil)
			if e != nil {
				log.Print("error creating profiler, ", err.Error())
			}
		}()
	}
	return &config
}
