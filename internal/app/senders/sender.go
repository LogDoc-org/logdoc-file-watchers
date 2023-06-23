package senders

import (
	"context"
	"file-watcher/internal/logdoc"
	"file-watcher/internal/structs"
	"log"
	"net"
	"sync"
	"time"
)

type LogDocSender struct {
	ctx          context.Context
	wg           *sync.WaitGroup
	LogDocConfig *structs.LD
	WatchingFile *structs.File
	LogDocStruct *logdoc.LogDocStruct
	Data         []byte
}

func New(ctx context.Context, wg *sync.WaitGroup, logDocConfig *structs.LD, watchingFile *structs.File, logDocStruct *logdoc.LogDocStruct, data []byte) *LogDocSender {
	return &LogDocSender{
		ctx:          ctx,
		wg:           wg,
		LogDocConfig: logDocConfig,
		WatchingFile: watchingFile,
		LogDocStruct: logDocStruct,
		Data:         data,
	}
}

func (s *LogDocSender) SendMessage(ldConnection *logdoc.LDConnection, ldConnCh chan net.Conn) {
	defer func() {
		s.wg.Done()
		// log.Println("<< Exiting LogDoc sender goroutine")
	}()

	for {
		log.Print(">> Sender for file ", s.WatchingFile.Path, " working...")
		select {
		case <-s.ctx.Done():
			return
		default:
			if ldConnection.Conn == nil {
				log.Println(s.WatchingFile.Path, " ERROR sending message to LogDoc, waiting for connection...")
				time.Sleep(2 * time.Second)
				continue
			}

			// Отправляем сообщение в LogDoc
			err := logdoc.SendMessage(*ldConnection.Conn, s.Data)
			if err != nil {
				if !ldConnection.MX.TryLock() {
					continue
				}
				ldConnection.Conn = nil
				// log.Println(s.WatchingFile.Path, " got lock and made connection empty")
				ldConnection.MX.Unlock()
				ldConnCh <- nil
				log.Println(s.WatchingFile.Path, " ERROR sending message to LogDoc, retrying...\n\terror: ", err)
				time.Sleep(time.Second)
				continue
			}

			log.Println(s.WatchingFile.Path, " message successfully sent to LogDoc")
			return
		}
	}
}
