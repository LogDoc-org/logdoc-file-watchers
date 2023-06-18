package watchers

import (
	"context"
	"file-watcher/internal/app/readers"
	"file-watcher/internal/app/structs"
	"net"
	"os"
	"sync"
	"time"
)

func WatchFile(ctx context.Context, wg *sync.WaitGroup, ldConf structs.LD, ldConnection *net.Conn, watchingFile structs.File) {
	defer func() {
		wg.Done()
		//log.Println("<< Exiting watcher goroutine WatchFile ", watchingFile.Path)
	}()

	// waiting for file
	for {
		select {
		case <-ctx.Done():
			return
		default:
			file, e := os.Open(watchingFile.Path)
			if e == nil {
				wg.Add(1)
				go readers.ReadFile(ctx, wg, ldConnection, &ldConf, &watchingFile, file)
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}
