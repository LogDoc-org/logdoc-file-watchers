package watchers

import (
	"bufio"
	"context"
	"file-watcher/internal/app/senders"
	"file-watcher/internal/logdoc"
	"file-watcher/internal/structs"
	"file-watcher/internal/utils"
	"github.com/fsnotify/fsnotify"
	"github.com/vjeantet/grok"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

func WatchFile(ctx context.Context, mx *sync.RWMutex, wg *sync.WaitGroup, grok *grok.Grok, ldConfig structs.LD, ldConnection *net.Conn, watchingFile structs.File) {
	defer func() {
		wg.Done()
		// log.Println("<< Exiting watcher goroutine WatchFile ", watchingFile.Path)
	}()

	if watchingFile.Custom != "" {
		err := grok.AddPattern("CUSTOM_DATE", watchingFile.Custom)
		if err != nil {
			log.Println("Error adding CUSTOM_DATE pattern, ", err)
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println(watchingFile.Path, " ошибка создания наблюдателя, ", err)
		return
	}
	defer watcher.Close()

	if strings.LastIndex(watchingFile.Path, "/") == -1 {
		err = watcher.Add(".")
		if err != nil {
			log.Println(watchingFile.Path, " watcher ERROR, ", err)
			return
		}
	} else {
		err = watcher.Add(watchingFile.Path[:strings.LastIndex(watchingFile.Path, "/")+1])
		if err != nil {
			log.Println(watchingFile.Path, " watcher ERROR, ", err)
			return
		}
	}

	// пытаемся открыть файл, если он уже есть
	file, err := os.Open(watchingFile.Path)
	if err != nil {
		log.Println(watchingFile.Path, " ERROR opening file! watching...")
	}

	if file != nil {
		err = rePositioning(file)
		if err != nil {
			log.Println(watchingFile.Path, " repositioning error, exiting! error:", err)
			return
		}
		log.Println(file.Name(), " готов! запускаем обработку...")
	}

	// waiting for file
	for {
		select {
		case <-ctx.Done():
			log.Println(watchingFile.Path, " получил сигнал остановки, завершаем наблюдение")
			return
		case event := <-watcher.Events:
			switch {
			case event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Chmod == fsnotify.Chmod:
				break
			case event.Op&fsnotify.Remove == fsnotify.Remove && event.Name == watchingFile.Path:
				log.Println(watchingFile.Path, " был удален, запускаем наблюдение...")
				file = nil
			case event.Op&fsnotify.Rename == fsnotify.Rename && event.Name == watchingFile.Path:
				log.Println(watchingFile.Path, " был переименован, запущен процесс log rotation? запускаем наблюдение...")
				file = nil
			case event.Op&fsnotify.Create == fsnotify.Create && event.Name == watchingFile.Path:
				file, err = os.Open(event.Name)
				if err != nil {
					log.Println(watchingFile.Path, " watcher ERROR, ошибка при попытке открыть файл для чтения, ", err)
					return
				}
				log.Println(file.Name(), " готов! запускаем обработку...")
			default:
				if strings.HasPrefix(event.Name, watchingFile.Path) && !strings.HasSuffix(event.Name, "~") {
					log.Println(watchingFile.Path, " watcher, got fs event:", event)
				}
			}
		case err = <-watcher.Errors:
			log.Println(watchingFile.Path, " watcher loop ERROR, ", err)
			return
		default:
			if file != nil {
				// Ошибок нет, читаем файл
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					data := scanner.Text()
					var logDocMessage []byte
					if data != "" {
						// Формируем LD структуру на основе текущей конфигурации
						logDocStruct := logdoc.LogDocStruct{
							Conn:              ldConnection,
							App:               utils.ProcessField(&ldConfig, &watchingFile, "app"),
							Src:               utils.ProcessField(&ldConfig, &watchingFile, "src"),
							Lvl:               utils.ProcessField(&ldConfig, &watchingFile, "lvl"),
							DateLayout:        watchingFile.Layout,
							CustomDatePattern: watchingFile.Custom,
						}
						for _, pattern := range watchingFile.Patterns {
							log.Println(watchingFile.Path, " trying pattern: ", pattern, "\n\tfile:", watchingFile.Path, "\n\tdata:", data)
							mx.Lock()
							logDocMessage, err = logDocStruct.ConstructMessageWithFields(grok, data, pattern)
							mx.Unlock()
							if err == nil {
								break
							}
							log.Println(watchingFile.Path, "ERROR, failed constructing LogDoc message:\n\tfile:", watchingFile.Path, "\n\tdata:", data, "\n\tpattern:", pattern, "\n\terror:", err, ", trying next pattern (if available)...")
						}
						// log.Println("LogDoc Message constructed, ready for sending, source date/time:", srcDateTime, ", data:", message)

						if logDocMessage != nil {
							wg.Add(1)
							sender := senders.New(ctx, wg, &ldConfig, &watchingFile, &logDocStruct, logDocMessage)
							go sender.SendMessage()
						}
						log.Println(watchingFile.Path, " patterns trying failed! Dropping message...\n\tfile:", watchingFile.Path, "\n\tdata:", data)
					}
				}

				if err = scanner.Err(); err != nil {
					log.Println(watchingFile.Path, " watcher scan error, ", err)
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func rePositioning(file io.Seeker) error {
	// перемещаем указатель файла на конец файла
	_, err := file.Seek(0, 2)
	if err != nil {
		return err
	}
	return nil
}
