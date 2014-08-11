package main

import (
	"io"
	"os"
	"fmt"
	"log"
	"flag"
	"net/http"
	"github.com/fzzy/radix/redis"
	"code.google.com/p/go.net/websocket"	
)

type T struct {
	Msg   string
	Count int
}

var (
	redis_addr = flag.String("redis", ":6379", "redis addr")
	port  = flag.String("port", "8080", "port to run the server on")
)

func PollRedis() chan string {
	ch := make(chan string)
	go func() {
		redis_conn, err := redis.Dial("tcp", *redis_addr)
		if err != nil {
			log.Fatal(err)
			return
		}
		for {
			key_element, err := redis_conn.Cmd("blpop", "data", 0).List()
			log.Print(fmt.Sprintf("Received: %v", key_element))
			if err != nil{
				log.Fatal(err)
			}
			ch <- key_element[1]
		}
		defer redis_conn.Close()
	}()
	return ch
}

func HttpServer(w http.ResponseWriter, r *http.Request){
	file, _ := os.Open("test.html")
	defer file.Close()
	io.Copy(w, file)
}

func WsServer(ws *websocket.Conn) {
	log.Print("Client connected")
	ch := PollRedis()
	for {
		data := <-ch
		log.Print(fmt.Sprintf("Sending to client %s", data))
		io.WriteString(ws, data)
	}
}

func main() {
	flag.Parse()
	http.HandleFunc("/test", HttpServer)
	http.Handle("/data", websocket.Handler(WsServer))
	log.Print(fmt.Sprintf("Running push server on %s...", *port))
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
