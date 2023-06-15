package main

import (
	"context"
	configreader "file-watcher/internal/config"
	"file-watcher/internal/logdoc"
	"file-watcher/internal/pkg/app"
	"file-watcher/internal/utils"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
)

func main() {
	utils.CreatePID()
	defer func() {
		err := os.Remove("RUNNING_PID")
		if err != nil {
			log.Fatal("Error removing PID file. Exiting...")
		}
	}()

	config := configreader.ProcessConfig()

	conn, e := logdoc.Connect(&config.LogDoc)
	if e != nil {
		//log.Fatal("Нет связи с LogDoc сервером, ", e)
	}

	// Gracefully Shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	// Для завершения работы горутин
	ctx, cancel := context.WithCancel(context.Background())

	// Для ожидания завершения работы всех горутин
	wg := sync.WaitGroup{}

	application := app.New(config, conn)

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		//log.Println("Got config file change:", e.Name, ", shutting down watchers, count:", atomic.LoadInt64(&application.Watchers))
		log.Println("Got config file change:", e.Name, ", shutting down watchers")

		cancel()
		wg.Wait()

		application.Mx.Lock()
		err := viper.Unmarshal(config)
		if err != nil {
			log.Fatal("Error parsing configuration")
		}
		application.FilesMap = make(map[string]string)
		application.Mx.Unlock()

		// Пересоздаем контекст, тк предыдущий уже отменен
		ctx, cancel = context.WithCancel(context.Background())
		wg.Add(1)

		go application.Run(ctx, &wg)
		fmt.Println("All goroutines restarted!")
	})

	log.Println("Application running, Press Ctrl+C to stop")
	wg.Add(1)
	go application.Run(ctx, &wg)

	<-sig
	log.Println("!! Got bye bye signal, shutting down watchers")

	cancel()
	wg.Wait()

	close(sig)
	log.Println("Application gracefully shutdown")
}
