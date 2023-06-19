package watchers

import (
	"context"
	"file-watcher/internal/app/readers"
	"file-watcher/internal/app/structs"
	"github.com/vjeantet/grok"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

func WatchFile(ctx context.Context, wg *sync.WaitGroup, grok *grok.Grok, ldConf structs.LD, ldConnection *net.Conn, watchingFile structs.File) {
	defer func() {
		wg.Done()
		//log.Println("<< Exiting watcher goroutine WatchFile ", watchingFile.Path)
	}()

	if watchingFile.Custom != "" {
		err := grok.AddPattern("CUSTOM_DATE", watchingFile.Custom)
		if err != nil {
			log.Println("Error adding CUSTOM_DATE pattern, ", err)
		}
	}

	// waiting for file
	for {
		select {
		case <-ctx.Done():
			return
		default:
			file, e := os.Open(watchingFile.Path)
			if e == nil {
				wg.Add(1)
				go readers.ReadFile(ctx, wg, grok, ldConnection, &ldConf, &watchingFile, file)
				return
			}
			time.Sleep(500 * time.Millisecond)
			//log.Print(">> Watcher ", watchingFile.Path, " working...")
		}
	}
}
