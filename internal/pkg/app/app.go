package app

import (
	"context"
	"file-watcher/internal/app/watchers"
	"file-watcher/internal/logdoc"
	"file-watcher/internal/structs"
	"file-watcher/internal/utils"
	"fmt"
	"github.com/vjeantet/grok"
	"log"
	"net"
	"runtime"
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

func (a *App) Run(ctx context.Context, shutdown chan struct{}, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		log.Println("<< Exiting Application")
	}()

	ldConnCh := make(chan net.Conn)

	ldConnection := logdoc.LDConnection{
		MX:   &sync.Mutex{},
		Conn: a.LogDocConnection,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case conn := <-ldConnCh:
				// поступил сигнал обрыва соединения
				if conn == nil {
					ldConnection.Conn = logDocReconnect(shutdown, a.Config)
				}
			default:
				time.Sleep(time.Second)
			}
		}
	}()

	if a.Config.Debug {
		wg.Add(1)
		go func() {
			defer wg.Done()

			var mem runtime.MemStats
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Minute):
					// Увлекаться этим сильно не надо, там stop / start the world
					runtime.ReadMemStats(&mem)
					fmt.Printf("Memstats:\n\tAlloc = %v MiB\n\tGoRoutines = %d\n", mem.Alloc/1024/1024, runtime.NumGoroutine())
				}
			}
		}()
	}

	g, err := grok.NewWithConfig(&grok.Config{NamedCapturesOnly: true})
	if err != nil {
		log.Panic("Error initializing patterns processor, ", err)
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
					wg.Add(1)
					go watchers.WatchFile(ctx, &a.Mx, wg, a.Grok, a.Config.LogDoc, &ldConnection, watchingFile, ldConnCh)
				}
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func logDocReconnect(shutdown chan struct{}, config *structs.Config) *net.Conn {
	// коннектимся к ЛД
	conn, err := utils.Retryer(func() (*net.Conn, error) {
		r, e := logdoc.Connect(&config.LogDoc)
		if e != nil {
			return nil, e
		}
		return r, nil
	}, config.LogDoc.Retries, 5*time.Second)
	if err != nil {
		log.Print("LogDoc connection failed, shutting down application")
		shutdown <- struct{}{}
		return nil
	}
	return conn
}
