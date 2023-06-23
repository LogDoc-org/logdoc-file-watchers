package logdoc

import (
	"bytes"
	"file-watcher/internal/structs"
	"fmt"
	"github.com/vjeantet/grok"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type LogDocStruct struct {
	DateLayout        string
	CustomDatePattern string
	App               string
	Src               string
	Lvl               string
	tsrc              string
	ip                string
	pid               string
}

func Connect(ldConf *structs.LD) (*net.Conn, error) {
	address := ldConf.Host + ":" + ldConf.Port

	log.Println("Connecting LogDoc server ", address, "...")
	conn, err := net.DialTimeout(ldConf.Proto, address, 5*time.Second)
	if err != nil {
		log.Println("Error connecting LogDoc server, ", address, err)

		return nil, err
	}
	log.Println("LogDoc server successfully connected")

	return &conn, nil
}

func SendMessage(conn io.Writer, msg []byte) error {
	_, err := conn.Write(msg)
	if err != nil {
		log.Println(fmt.Errorf("SendMessage ERROR, ошибка записи в соединение, %w", err))
		return err
	}
	return nil
}

func PrepareLogDocMessage(conn *net.Conn, ld *LogDocStruct, srcDateTime string, message string) ([]byte, error) {
	if conn == nil {
		log.Println("Error in SendMessage, connection not available")
		return nil, fmt.Errorf("error in SendMessage, connection not available")
	}

	var lvl string
	if strings.Compare(ld.Lvl, "warning") == 0 {
		lvl = "warn"
	} else {
		lvl = ld.Lvl
	}

	ip := (*conn).RemoteAddr().String()
	pid := fmt.Sprintf("%d", os.Getpid())
	src := ld.Src

	t, err := time.Parse(ld.DateLayout, srcDateTime)
	if err != nil {
		log.Println("Error parsing source date time:\n\t", srcDateTime, "\n\tlayout:", ld.DateLayout)
		return nil, err
	}
	tsrc := t.Format("060102150405.000") + "\n"

	// Пишем заголовок
	result := []byte{6, 3}
	// Записываем само сообщение
	WritePair("msg", extractMessage(message), &result)
	// Обрабатываем кастомные поля
	ProcessCustomFields(message, &result)
	// Служебные поля
	WritePair("app", ld.App, &result)
	WritePair("tsrc", tsrc, &result)
	WritePair("lvl", lvl, &result)
	WritePair("ip", ip, &result)
	WritePair("pid", pid, &result)
	WritePair("src", src, &result)

	// Финальный байт, завершаем
	result = append(result, []byte("\n")...)

	if *conn == nil {
		log.Println(fmt.Errorf("соединение c LogDoc сервером потеряно, %w", err))
		return nil, err
	}

	return result, nil
}

func WritePair(key string, value string, arr *[]byte) {
	sepIdx := strings.Index(value, "@@")
	msg := ""
	if sepIdx != -1 {
		msg = value[0:sepIdx]
	} else {
		msg = value
	}
	if strings.Contains(msg, "\n") {
		writeComplexPair(key, msg, arr)
	} else {
		writeSimplePair(key, msg, arr)
	}
}

func (ld *LogDocStruct) ConstructMessageWithFields(ldConnection *net.Conn, g *grok.Grok, message string, pattern string) ([]byte, error) {
	var ldMessage strings.Builder
	ldMessage.WriteString(message)

	values, err := g.Parse(pattern, message)
	if err != nil {
		log.Println("Error parsing message, ", err)
		return nil, err
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("pattern error, no values available")
	}

	var i int
	for key, val := range values {
		if i == 0 {
			ldMessage.WriteString(fmt.Sprintf("@@%s=%s", key, val))
		} else {
			ldMessage.WriteString(fmt.Sprintf("@%s=%s", key, val))
		}
		i++
	}

	if _, ok := values["timestamp"]; !ok {
		values["timestamp"] = "01/Jun/1951:23:59:59 +0300"
	}

	data, err := PrepareLogDocMessage(ldConnection, ld, values["timestamp"], ldMessage.String())
	if err != nil {
		log.Println("Error Preparing LogDoc Message Structure:\n\tdata source date/time:", values["timestamp"], "\n\tmessage:", ldMessage.String())
		return nil, err
	}

	return data, nil
}

func writeComplexPair(key string, value string, arr *[]byte) {
	*arr = append(*arr, []byte(key)...)
	*arr = append(*arr, []byte("\n")...)
	*arr = append(*arr, writeInt(len(value))...)
	*arr = append(*arr, []byte(value)...)
}

func writeSimplePair(key string, value string, arr *[]byte) {
	*arr = append(*arr, []byte(key+"="+value+"\n")...)
}

func ProcessCustomFields(msg string, arr *[]byte) {
	// Обработка кастом полей
	sepIdx := strings.Index(msg, "@@")
	rawFields := ""

	if sepIdx != -1 {
		rawFields = msg[sepIdx+2:]
		keyValuePairs := strings.Split(rawFields, "@")

		for _, pair := range keyValuePairs {
			keyValue := strings.Split(pair, "=")
			if len(keyValue) == 2 && (keyValue[0] != "message" && keyValue[0] != "timestamp") {
				*arr = append(*arr, []byte(keyValue[0]+"="+keyValue[1]+"\n")...)
			}
		}
	}
}

func extractMessage(msg string) string {
	var message string
	// Обработка кастом полей
	sepIdx := strings.Index(msg, "@@")
	rawFields := ""

	if sepIdx != -1 {
		rawFields = msg[sepIdx+2:]
		keyValuePairs := strings.Split(rawFields, "@")

		for _, pair := range keyValuePairs {
			keyValue := strings.Split(pair, "=")
			if len(keyValue) >= 2 {
				if keyValue[0] == "message" && keyValue[1] != "" {
					for i := range keyValue {
						if i > 0 {
							message = message + keyValue[i] + "="
						}
					}
					message = message[:len(message)-1]
					break
				}
			}
		}
	}
	if message == "" {
		return msg
	}
	return message
}

func writeInt(in int) []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(byte((in >> 24) & 0xff))
	buf.WriteByte(byte((in >> 16) & 0xff))
	buf.WriteByte(byte((in >> 8) & 0xff))
	buf.WriteByte(byte(in & 0xff))
	return buf.Bytes()
}
