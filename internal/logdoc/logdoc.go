package logdoc

import (
	"bytes"
	"file-watcher/internal/app/structs"
	"fmt"
	"github.com/vjeantet/grok"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type LogDocStruct struct {
	Conn              *net.Conn
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

func (ld *LogDocStruct) SendMessage(srcDateTime string, message string) error {

	if ld.Conn == nil {
		log.Println("Error in SendMessage, connection not available")
		return fmt.Errorf("error in SendMessage, connection not available")
	}

	var lvl string
	if strings.Compare(ld.Lvl, "warning") == 0 {
		lvl = "warn"
	} else {
		lvl = ld.Lvl
	}

	ip := (*ld.Conn).RemoteAddr().String()
	pid := fmt.Sprintf("%d", os.Getpid())
	src := ld.Src

	t, err := time.Parse(ld.DateLayout, srcDateTime)
	if err != nil {
		log.Println("Error parsing source date:", srcDateTime)
		return err
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

	if *ld.Conn == nil {
		log.Println(fmt.Errorf("соединение c LogDoc сервером потеряно, %w", err))
		return err
	}

	_, err = (*ld.Conn).Write(result)
	if err != nil {
		log.Println(fmt.Errorf("ошибка записи в соединение, %w", err))
		return err
	}
	return nil
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

func (ld *LogDocStruct) ConstructMessageWithFields(message string, pattern string) (string, string, error) {
	var ldMessage strings.Builder
	ldMessage.WriteString(message)

	g, _ := grok.NewWithConfig(&grok.Config{NamedCapturesOnly: true})

	if ld.CustomDatePattern != "" {
		err := g.AddPattern("CUSTOM_DATE", ld.CustomDatePattern)
		if err != nil {
			return "", "", err
		}
	}

	values, err := g.Parse(pattern, message)
	if err != nil {
		log.Println("Error parsing message, ", err)
		return "", "", err
	}

	if len(values) == 0 {
		return "", "", fmt.Errorf("pattern error, no values available")
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
	return values["timestamp"], ldMessage.String(), nil
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
			if len(keyValue) == 2 {
				if keyValue[0] == "message" && keyValue[1] != "" {
					message = keyValue[1]
				}
			}
		}
	}
	if message == "" {
		return msg
	} else {
		return message
	}
}

func writeInt(in int) []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(byte((in >> 24) & 0xff))
	buf.WriteByte(byte((in >> 16) & 0xff))
	buf.WriteByte(byte((in >> 8) & 0xff))
	buf.WriteByte(byte(in & 0xff))
	return buf.Bytes()
}

func GetSourceName(pc uintptr, file string, line int, ok bool) string {
	// in skip if we're using 1, so it will actually log the where the error happened, 0 = this function
	return file[strings.LastIndex(file, "/")+1:]
}

func GetSourceLineNum(pc uintptr, file string, line int, ok bool) int {
	// in skip if we're using 1, so it will actually log the where the error happened, 0 = this function
	return line
}
