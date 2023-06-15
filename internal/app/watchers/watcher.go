package watchers

import (
	"context"
	"file-watcher/internal/app/readers"
	"file-watcher/internal/app/structs"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

func WatchFile(ctx context.Context, wg *sync.WaitGroup, ldConf structs.LD, ldConnection *net.Conn, configFile structs.File, filename string) {
	defer func() {
		wg.Done()
		log.Println("<< Exiting watcher goroutine WatchFile ", filename)
	}()

	// waiting for file
	for {
		select {
		case <-ctx.Done():
			return
		default:
			file, e := os.Open(filename)
			if e == nil {
				wg.Add(1)
				go readers.ReadFile(ctx, wg, ldConnection, &ldConf, &configFile, file)
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}
