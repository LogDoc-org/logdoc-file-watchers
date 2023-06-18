package utils

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

func Ternary(condition bool, iftrue, iffalse any) any {
	if condition {
		return iftrue
	}
	return iffalse
}

// реализация паттерна timeout and retry
func Retryer(fn func() (*net.Conn, error), maxRetries int, initialRetryInterval time.Duration) (*net.Conn, error) {
	if maxRetries == 0 {
		i := 0
		for {
			resp, err := fn()
			if err == nil {
				return resp, nil
			}
			log.Println(fmt.Errorf("unable to call function, attempt %d ", i+1))
			i++
			time.Sleep(initialRetryInterval)
		}
	} else {

		for i := 0; i < maxRetries; i++ {
			resp, err := fn()
			if err == nil {
				return resp, nil
			}
			log.Println(fmt.Errorf("unable to call function, attempt %d ", i+1))
			time.Sleep(initialRetryInterval)
		}
		return nil, fmt.Errorf("unable to complete task after %d attempts", maxRetries)
	}
}

func CreatePID() {
	// Сохраним id запущенного процесса в файл
	pid := os.Getpid()
	err := os.WriteFile("RUNNING_PID", []byte(strconv.Itoa(pid)), 0600)
	if err != nil {
		log.Fatal("Error writing PID to file. Exiting...")
	}
	log.Println("Service RUNNING PID created")
}
