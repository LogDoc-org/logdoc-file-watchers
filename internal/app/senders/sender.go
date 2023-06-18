package senders

import (
	"context"
	"file-watcher/internal/app/structs"
	"file-watcher/internal/logdoc"
	"file-watcher/internal/utils"
	"log"
	"net"
	"sync"
	"time"
)

func LogDocSender(ctx context.Context, wg *sync.WaitGroup, ldConf *structs.LD, ld *logdoc.LogDocStruct, srcDateTime string, message string) {
	defer func() {
		wg.Done()
		log.Println("<< Exiting LogDoc sender goroutine")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if ld.Conn == nil {
				// коннектимся к ЛД
				conn, err := utils.Retryer(func() (*net.Conn, error) {
					r, e := logdoc.Connect(ldConf)
					if e != nil {
						return nil, e
					}
					return r, nil
				}, 3, 5*time.Second)
				if err != nil {
					//log.Panic("LogDoc connection failed")
					log.Println("LogDoc connection failed")
				}
				ld.Conn = conn
			}

			// Отправляем сообщение в LogDoc
			err := logdoc.SendMessage(*ld.Conn, ld, srcDateTime, message)
			if err == nil {
				log.Println("Message successfully sent to LogDoc:\n\tdata source date/time:", srcDateTime, "\n\tmessage:", message)
				return
			}

			log.Println("ERROR sending message to LogDoc, reconnecting, error: ", err)

			ld.Conn = nil
		}
		time.Sleep(time.Second)
	}
}
