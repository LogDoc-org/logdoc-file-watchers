package readers

import (
	"bufio"
	"context"
	"file-watcher/internal/app/senders"
	"file-watcher/internal/app/structs"
	"file-watcher/internal/logdoc"
	"file-watcher/internal/utils"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

func ReadFile(ctx context.Context, wg *sync.WaitGroup, ldConnection *net.Conn, ldConfig *structs.LD, configFile *structs.File, file *os.File) {
	defer func() {
		file.Close()
		wg.Done()
		log.Println("<< Exiting reader goroutine ReadFile ", file.Name())
	}()

	// Формируем LD структуру на основе текущей конфигурации
	logDocStruct := logdoc.LogDocStruct{
		Conn:              ldConnection,
		App:               processField(ldConfig, configFile, "app"),
		Src:               processField(ldConfig, configFile, "src"),
		Lvl:               processField(ldConfig, configFile, "lvl"),
		DateLayout:        configFile.Layout,
		CustomDatePattern: configFile.Custom,
	}

	log.Println("File ", file.Name(), " ready! Reading...")

	// перемещаем указатель файла на конец файла
	_, err := file.Seek(0, 2)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if _, e := os.Stat(file.Name()); os.IsNotExist(e) {
				fmt.Println("File ", file.Name(), " does not exist")
				return
			}

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				data := scanner.Text()
				var logDocMessage []byte
				if data != "" {
					for _, pattern := range configFile.Patterns {
						logDocMessage, err = logDocStruct.ConstructMessageWithFields(data, pattern)
						if err == nil {
							break
						}
						log.Println("Error constructing LogDoc message:\n\tdata:", data, "\n\tpattern:", pattern, "\n\terror:", err, ", trying next pattern (if available)...")
					}
					//log.Println("LogDoc Message constructed, ready for sending, source date/time:", srcDateTime, ", data:", message)
					wg.Add(1)
					sender := senders.New(ctx, wg, ldConfig, &logDocStruct, logDocMessage)
					go sender.SendMessage()
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func processField(ldConf *structs.LD, configFile *structs.File, field string) string {
	switch field {
	case "app":
		return utils.Ternary(configFile.App == "", ldConf.Default.App, configFile.App).(string)
	case "src":
		return utils.Ternary(configFile.Source == "", ldConf.Default.Source, configFile.Source).(string)
	case "lvl":
		return utils.Ternary(configFile.Level == "", ldConf.Default.Level, configFile.Level).(string)
	default:
		return ""
	}
}
