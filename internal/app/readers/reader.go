package readers

import (
	"bufio"
	"context"
	"file-watcher/internal/app/senders"
	"file-watcher/internal/app/structs"
	"file-watcher/internal/logdoc"
	"file-watcher/internal/utils"
	"github.com/vjeantet/grok"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

func ReadFile(ctx context.Context, wg *sync.WaitGroup, g *grok.Grok, ldConnection *net.Conn, ldConfig *structs.LD, watchingFile *structs.File, file *os.File) {
	defer func() {
		file.Close()
		wg.Done()
		log.Println("<< Exiting reader goroutine ReadFile ", file.Name())
	}()

	// Формируем LD структуру на основе текущей конфигурации
	logDocStruct := logdoc.LogDocStruct{
		Conn:              ldConnection,
		App:               processField(ldConfig, watchingFile, "app"),
		Src:               processField(ldConfig, watchingFile, "src"),
		Lvl:               processField(ldConfig, watchingFile, "lvl"),
		DateLayout:        watchingFile.Layout,
		CustomDatePattern: watchingFile.Custom,
	}

	log.Println("File ", file.Name(), " ready! Reading...")

	// перемещаем указатель файла на конец файла
	err := rePositioning(file)
	if err != nil {
		return
	}

	var prevFileSize int64

	for {
		select {
		case <-ctx.Done():
			return
		default:

			if file == nil {
				file, err = os.Open(watchingFile.Path)
				if err != nil {
					//log.Println("Error opening file ", file.Name())
					time.Sleep(500 * time.Millisecond)
					continue
				}
				err := rePositioning(file)
				if err != nil {
					log.Println("ERROR: Ошибка перепозиционирования по файлу ", file.Name(), " после его усечения, Выходим...")
					return
				}
				log.Println("File ", file.Name(), " ready! Reading...")
				fileInfo, _ := os.Stat(file.Name())
				prevFileSize = fileInfo.Size()
			} else {
				fileInfo, e := os.Stat(file.Name())
				if os.IsNotExist(e) {
					log.Println("File ", file.Name(), " does not exists! waiting for file...")
					file, err = os.Open(watchingFile.Path)
					if err != nil {
						//log.Println("Error opening file ", file.Name())
						time.Sleep(500 * time.Millisecond)
						continue
					}
					err := rePositioning(file)
					if err != nil {
						log.Println("ERROR: Ошибка перепозиционирования по файлу ", file.Name(), " после его усечения, Выходим...")
						return
					}
					log.Println("File ", file.Name(), " ready! Reading...")
					fileInfo, _ = os.Stat(file.Name())
					prevFileSize = fileInfo.Size()
				}
				if fileInfo.Size() < prevFileSize {
					log.Println("файл ", file.Name(), " был изменен в сторону уменьшения, переоткрываем")
					prevFileSize = fileInfo.Size()
					// перемещаем указатель файла на конец файла
					err := rePositioning(file)
					if err != nil {
						log.Println("ERROR: Ошибка перепозиционирования по файлу ", file.Name(), " после его усечения, Выходим...")
						return
					}
				}
			}

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				data := scanner.Text()
				var logDocMessage []byte
				if data != "" {
					for _, pattern := range watchingFile.Patterns {
						log.Println("Trying pattern: ", pattern, "\n\tfile:", watchingFile.Path, "\n\tdata:", data)
						logDocMessage, err = logDocStruct.ConstructMessageWithFields(g, data, pattern)
						if err == nil {
							break
						}
						log.Println("Error constructing LogDoc message:\n\tfile:", watchingFile.Path, "\n\tdata:", data, "\n\tpattern:", pattern, "\n\terror:", err, ", trying next pattern (if available)...")
					}
					//log.Println("LogDoc Message constructed, ready for sending, source date/time:", srcDateTime, ", data:", message)

					if logDocMessage == nil {
						log.Println("Patterns trying failed! Dropping message...\n\tfile:", watchingFile.Path, "\n\tdata:", data)
						goto CONTINUE
					}
					wg.Add(1)
					sender := senders.New(ctx, wg, ldConfig, watchingFile, &logDocStruct, logDocMessage)
					go sender.SendMessage()
				}
			}
		CONTINUE:
			time.Sleep(500 * time.Millisecond)
			//log.Print(">> Reader ", file.Name(), " working...")
			prevFileSize = fileInfo.Size()
		}
	}
}

func checkFile(file *os.File) {

}

func rePositioning(file *os.File) error {
	// перемещаем указатель файла на конец файла
	_, err := file.Seek(0, 2)
	if err != nil {
		log.Println("ERROR: Ошибка перемещения указателя, ", err)
		return err
	}
	return nil
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
