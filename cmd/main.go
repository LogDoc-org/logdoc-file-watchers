package main

import (
	"context"
	configreader "file-watcher/internal/config"
	"file-watcher/internal/logdoc"
	"file-watcher/internal/pkg/app"
	"file-watcher/internal/utils"
	"flag"
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

	cflag := flag.String("config", "", "-config=application.json")
	flag.Parse()
	config := configreader.ProcessConfig(*cflag)

	conn, e := logdoc.Connect(&config.LogDoc)
	if e != nil {
		log.Println(" >> Fatal Error: Нет связи с LogDoc сервером, ", e)
		return
	}
	defer (*conn).Close()

	shutdown := make(chan struct{})

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
		log.Println("Application config file is changing...\n\tfile name:", e.Name, "\nshutting down watchers...")

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
		go application.Run(ctx, shutdown, &wg)
		log.Println("All goroutines restarted!")
	})

	log.Println("Application running, Press Ctrl+C to stop")
	wg.Add(1)
	go application.Run(ctx, shutdown, &wg)

	select {
	case <-sig:
		log.Println("!! Got bye bye signal, shutting down watchers")
	case <-shutdown:
		log.Println("!! Got shutdown signal, shutting down watchers")
	}
	cancel()
	wg.Wait()

	close(sig)
	log.Println("Application gracefully shutdown")
}
