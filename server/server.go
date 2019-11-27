package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"../common"
	"../gouuid"
)

/*
const (
	port string = ":1200"
)
*/

var (
	//host = flag.String("host", "0.0.0.0", "host to listen on")
	iport = flag.Int("port", 1200, "port to listen on")
	blockSize = flag.Int("size", 1024, "block size to read packets on")
)

var p = fmt.Println
var pf = fmt.Printf

type Server struct {
	conn     *net.UDPConn
	messages chan string
	clients  map[*uuid.UUID]Client
}

type Client struct {
	userID   uuid.UUID
	userName string
	userAddr *net.UDPAddr
}

type Message struct {
	messageType      common.MessageType
	userID           *uuid.UUID
	userName         string
	content          string
	connectionStatus common.ConnectionStatus
	time             string
}

func (server *Server) handleMessage() {
	//var buf [blockSize]byte
	buf := make([]byte, *blockSize)

	n, addr, err := server.conn.ReadFromUDP(buf[0:])
	if err != nil {
		pf("ReadFromUDP error ")
		return
	}

	msg := string(buf[0:n])
	m := server.parseSimpleMessage(msg) //server.parseMessage(msg)

	if m.connectionStatus == common.LEAVING {
		delete(server.clients, m.userID)
		server.messages <- msg
		pf("%s left", m.userName)
	} else {
		switch m.messageType {
		case common.FUNC:
			var c Client
			c.userAddr = addr
			c.userID = *m.userID
			c.userName = m.userName
			server.clients[m.userID] = c
			server.messages <- msg
			pf("%s joining , addr=%s", m.userName, addr)
		case common.CLASSIQUE:

			pf("%s %s: %s", m.time, m.userName, m.content)
			server.messages <- msg
		default:
			// send back 
			ack_msg := m.content // fmt.Sprintf("server got :%s", m.content)
			pf("raw msg %s , ack_msg %s\n", m.content, ack_msg)
			//server.messages <- ack_msg
			server.conn.WriteToUDP([]byte(ack_msg), addr)
		}
	}
}

func (s *Server) parseSimpleMessage(msg string) (m Message) {
	m.content = msg 
	m.messageType = common.CUSTOM
	return
}


func (s *Server) parseMessage(msg string) (m Message) {

	stringArray := strings.Split(msg, "\x01")

	if len(stringArray) < 5 {
		m.content = msg
		return
	}

	fmt.Println("")
	m.userID, _ = uuid.ParseHex(stringArray[0])
	messageTypeStr, _ := strconv.Atoi(stringArray[1])
	m.messageType = common.MessageType(messageTypeStr)
	m.userName = stringArray[2]
	m.content = stringArray[3]
    m.time =  stringArray[4]
	//pf("MESSAGE RECEIVED: %s \n", msg)
	pf("USER NAME: %s \n", stringArray [2])
	pf("CONTENT: %s \n", stringArray [3])
	if strings.HasPrefix(m.content, ":q") || strings.HasPrefix(m.content, ":quit") {
		pf("%s is leaving \n", m.userName)
		m.connectionStatus = common.LEAVING
	}
	return
}

func (s *Server) sendMessage() {
	for {
		msg := <-s.messages
		//p(00, sendstr)
		for _, c := range s.clients {
			//pf("send %s , addr=%s \n", msg, c.userAddr)
			_, err := s.conn.WriteToUDP([]byte(msg), c.userAddr)
			//pf("Bytes read %d, error: %v", n, err)
			checkError(err)
		}
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error:%s", err.Error())
		os.Exit(1)
	}
}

func main() {	
	flag.Parse()

	port := fmt.Sprintf(":%d", *iport)
	fmt.Println("iport=%d,port=%s", *iport, port)

	udpAddress, err := net.ResolveUDPAddr("udp4", port)
	checkError(err)

	pf("IP=%s, Port %d \n", udpAddress.IP, udpAddress.Port)

	var s Server
	s.messages = make(chan string, 20)
	s.clients = make(map[*uuid.UUID]Client, 0)

	s.conn, err = net.ListenUDP("udp", udpAddress)
	checkError(err)

	go s.sendMessage()

	for {
		s.handleMessage()
	}
}
