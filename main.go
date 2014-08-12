package main

import (
	"code.google.com/p/go.net/websocket"
	"flag"
	"fmt"
	"github.com/fzzy/radix/redis"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
)

var (
	redis_addr  = flag.String("redis", ":6379", "redis addr")
	port        = flag.String("port", "8080", "port to run the server on")
	queue       = flag.String("queue", "default", "redis queue to listen to")
	subscribers = make([]chan string, 0)
	mutex       = &sync.Mutex{}
)

func Subscribe(ch chan string) (index int) {
	mutex.Lock()
	subscribers = append(subscribers, ch)
	index = len(subscribers) - 1
	mutex.Unlock()
	return
}

func Unsubscribe(index int) {
	mutex.Lock()
	subscribers[index] = nil
	mutex.Unlock()
}

func PollRedis() {
	redis_conn, err := redis.Dial("tcp", *redis_addr)
	if err != nil {
		log.Fatal(err)
	}
	defer redis_conn.Close()
	redis_conn.Cmd("subscribe", *queue)
	for {
		data, err := redis_conn.ReadReply().List()
		if err != nil {
			log.Fatal(err)
		}
		for _, subscriber := range subscribers {
			if subscriber != nil {
				subscriber <- data[2]
			}
		}
	}
}

func HttpServer(w http.ResponseWriter, r *http.Request) {
	file, _ := os.Open("test.html")
	defer file.Close()
	io.Copy(w, file)
}

func WsServer(ws *websocket.Conn) {
	ch := make(chan string)
	index := Subscribe(ch)
	log.Print(fmt.Sprintf("Client %d connected", index))
	defer func() {
		close(ch)
		Unsubscribe(index)
	}()
	for {
		data := <-ch
		_, err := io.WriteString(ws, data)
		if err != nil {
			log.Print(fmt.Sprintf("Client %d disconnected", index))
			break
		}
	}
}

func main() {
	flag.Parse()
	http.HandleFunc("/test", HttpServer)
	http.Handle("/data", websocket.Handler(WsServer))
	go PollRedis()
	log.Print(fmt.Sprintf("Running push server on %s...", *port))
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
