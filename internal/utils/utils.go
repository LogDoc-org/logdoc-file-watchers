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

// реализация паттерна timeout and retry, прогрессивный ретрайер
func Retryer(fn func() (*net.Conn, error), maxRetries int, initialRetryInterval time.Duration) (*net.Conn, error) {
	retryInterval := initialRetryInterval
	for i := 0; i < maxRetries; i++ {
		resp, err := fn()
		if err == nil {
			return resp, nil
		}
		log.Println(fmt.Errorf("unable to call function, attempt %d ", i+1))
		time.Sleep(retryInterval)
		//retryInterval *= 2 // увеличиваем время ожидания в 2 раза с каждой итерацией
	}
	return nil, fmt.Errorf("unable to complete task after %d attempts", maxRetries)
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
