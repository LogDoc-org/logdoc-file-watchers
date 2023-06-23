package senders

import (
	"context"
	"file-watcher/internal/logdoc"
	"file-watcher/internal/structs"
	"file-watcher/internal/utils"
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

func (s *LogDocSender) SendMessage() {
	defer func() {
		s.wg.Done()
		//log.Println("<< Exiting LogDoc sender goroutine")
	}()

	for {
		//log.Print(">> Sender for file ", s.WatchingFile.Path, " working...")
		select {
		case <-s.ctx.Done():
			return
		default:
			if s.LogDocStruct.Conn == nil {
				conn := s.LogDocReconnect()
				if conn == nil {
					continue
				}
				s.LogDocStruct.Conn = conn
			}

			// Отправляем сообщение в LogDoc
			err := logdoc.SendMessage(*s.LogDocStruct.Conn, s.Data)
			if err != nil {
				log.Println("ERROR sending message to LogDoc, retrying...\n\terror: ", err)
				time.Sleep(time.Second)
				conn := s.LogDocReconnect()
				s.LogDocStruct.Conn = conn
				continue
			}

			log.Println(s.WatchingFile.Path, " message successfully sent to LogDoc")
			return
		}
	}
}

func (s *LogDocSender) LogDocReconnect() *net.Conn {
	// коннектимся к ЛД
	conn, err := utils.Retryer(func() (*net.Conn, error) {
		r, e := logdoc.Connect(s.LogDocConfig)
		if e != nil {
			return nil, e
		}
		return r, nil
	}, s.LogDocConfig.Retries, 5*time.Second)
	if err != nil {
		//log.Panic("LogDoc connection failed")
		log.Println("LogDoc connection failed")
	}
	return conn
}
