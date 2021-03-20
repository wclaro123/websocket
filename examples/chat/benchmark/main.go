package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"strconv"
	"strings"
	"time"
)

const (
	format = "2006-01-02 15:04:05.999999999 -0700 MST"
	host   = "localhost"
	port   = "8080"
)

type Statistic struct {
	AvgDelay int
	Received int
	Sent     int
	Ratio    int
}

type Data struct {
	Number int
	Value  int
}

func main() {
	qty, duration, interval, err := config()
	if err != nil {
		return
	}
	var (
		send    = make(chan Data, qty)
		receive = make(chan Data, qty)
		avg     = make(chan Data, qty)
		start   = make(chan bool, qty)
	)

	for i := 0; i < qty; i++ {
		conn, err := createClient(host, port)
		if err != nil {
			fmt.Println(err)
		}

		go sendMessages(conn, i, interval, start, send)

		go receiveMessages(conn, i, receive, avg)

		go closeConn(conn, duration)
	}

	// start messaging after clients are ready
	for i := 0; i < qty; i++ {
		start <- true
	}

	resultMap := processResults(qty, send, receive, avg)

	printResults(qty, resultMap)
}

func readInput(msg string) (input string, err error) {
	fmt.Println(msg)
	_, err = fmt.Scanln(&input)
	return input, err
}

func config() (qty, duration, interval int, err error) {
	input, err := readInput("please enter the quantity of clients(max 247 in my computer): ")
	if err != nil {
		fmt.Println(err)
		return
	}

	qty, err = strconv.Atoi(input)
	if err != nil {
		fmt.Println("invalid quantity argument: ", input)
		return
	}
	if qty > 250 {
		fmt.Println("invalid quantity(max 247): ", input)
		return
	}

	input, err = readInput("please enter the duration in seconds of the test: ")
	if err != nil {
		fmt.Println(err)
		return
	}

	duration, err = strconv.Atoi(input)
	if err != nil {
		fmt.Println("invalid duration argument: ", input)
		return
	}

	input, err = readInput("please enter the time interval between messages in milliseconds : ")
	if err != nil {
		fmt.Println(err)
		return
	}

	interval, err = strconv.Atoi(input)
	if err != nil {
		fmt.Println("invalid duration argument: ", input)
		return
	}

	return
}

func createClient(host, port string) (conn *websocket.Conn, err error) {
	conn, _, err = websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s:%s/ws", host, port), nil)
	return
}

func closeConn(conn *websocket.Conn, duration int) {
	time.Sleep(time.Duration(duration) * time.Second)
	conn.Close()
}

func sendMessages(conn *websocket.Conn, i, interval int, start chan bool, send chan Data) {
	<-start

	qty := 0
	for {
		qty++
		time.Sleep(time.Duration(interval) * time.Millisecond)
		err := conn.WriteMessage(websocket.TextMessage, []byte(time.Now().Format(format)))
		if err != nil {
			send <- Data{
				Number: i,
				Value:  qty,
			}
			return
		}
	}
}

func receiveMessages(conn *websocket.Conn, i int, receive chan Data, avg chan Data) {
	var total int64
	var qty int
	for {
		_, m, err := conn.ReadMessage()
		if err != nil {
			break
		}

		messages := strings.Split(string(m), "\n")
		for _, message := range messages {
			t, err := time.Parse(format, message)
			if err != nil {
				fmt.Println("ERR:", err)
				break
			}
			n := time.Now()
			d := n.Sub(t)
			total += d.Nanoseconds()
			qty++
		}

	}
	receive <- Data{
		Number: i,
		Value:  qty,
	}
	avg <- Data{
		Number: i,
		Value:  int(total) / qty,
	}
}

func processResults(qty int, send, receive, avg chan Data) map[int]Statistic {
	resultMap := make(map[int]Statistic, qty)
	for i := 0; i < qty; i++ {
		s := <-send
		r := <-receive
		a := <-avg

		statistic := resultMap[s.Number]
		statistic.Sent = s.Value
		resultMap[s.Number] = statistic

		statistic = resultMap[r.Number]
		statistic.Received = r.Value
		resultMap[r.Number] = statistic

		statistic = resultMap[a.Number]
		statistic.AvgDelay = a.Value
		resultMap[a.Number] = statistic
	}
	return resultMap
}

func printResults(qty int, resultMap map[int]Statistic) {
	general := Statistic{}
	for i, v := range resultMap {
		fmt.Println("Client number: ", i)
		fmt.Println("Average delay: ", time.Duration(v.AvgDelay).String())
		fmt.Println("Messages send: ", v.Sent)
		fmt.Println("Messages received: ", v.Received)
		general.Sent += v.Sent
		general.Received += v.Received
		general.AvgDelay += v.AvgDelay
	}
	general.AvgDelay /= qty
	fmt.Println("------------------------------")
	fmt.Println("Average delay: ", time.Duration(general.AvgDelay).String())
	fmt.Println("Messages send: ", general.Sent)
	fmt.Println("Messages received: ", general.Received)
	fmt.Println("Ratio of messages lost: ", float64(general.Received)/float64(qty)/float64(general.Sent)*100)
}

