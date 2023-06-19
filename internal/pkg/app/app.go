package app

import (
	"context"
	"file-watcher/internal/app/structs"
	"file-watcher/internal/app/watchers"
	"github.com/vjeantet/grok"
	"log"
	"net"
	"sync"
	"time"
)

type App struct {
	Mx               sync.RWMutex
	Config           *structs.Config
	LogDocConnection *net.Conn
	Grok             *grok.Grok

	FilesMap map[string]string
	//Watchers int64
}

func New(config *structs.Config, conn *net.Conn) *App {
	return &App{
		Mx:               sync.RWMutex{},
		Config:           config,
		FilesMap:         make(map[string]string),
		LogDocConnection: conn,
		//Watchers: 0,
	}
}

func (a *App) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		log.Println("<< Exiting Application")
	}()

	g, err := grok.NewWithConfig(&grok.Config{NamedCapturesOnly: true})
	if err != nil {
		log.Fatal("Error initializing patterns processor, ", err)
	}
	a.Grok = g

	// крутимся бесконечно
	// может измениться конфиг
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Перебираем все указанные файлы
			// здесь могут появляться новые, удаляться старые
			a.Mx.RLock()
			size := len(a.Config.Files)
			a.Mx.RUnlock()

			for i := 0; i < size; i++ {
				a.Mx.RLock()
				filename := a.Config.Files[i].Path
				_, ok := a.FilesMap[filename]
				a.Mx.RUnlock()

				if filename == "" {
					log.Panic("error reading path from config file")
				}

				if !ok {
					log.Println("Waiting for file ", filename, "...")

					// добавляем файл в мапу, чтобы больше не стартовать его watcher
					a.Mx.Lock()
					a.FilesMap[filename] = "waiting"
					a.Mx.Unlock()

					var watchingFile structs.File
					for _, val := range a.Config.Files {
						if val.Path == filename {
							watchingFile = val
						}
					}
					//atomic.AddInt64(&a.Watchers, 1)
					wg.Add(1)
					go watchers.WatchFile(ctx, wg, a.Grok, a.Config.LogDoc, a.LogDocConnection, watchingFile)
				}
			}
			time.Sleep(1 * time.Second)
		}
	}
}
