package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dedis/protobuf"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
)
var server *ServerInfo
var nickName string = "John Doe"
var encrypted bool = false
var lock sync.Mutex

type ServerInfo struct {
	Addr *net.UDPAddr
	MsgBuffer []peerMessage
	Peers []string
}

type updateMessage struct {
	Messages []peerMessage
}

type setupMessage struct {
	Addr string
	Peers []string
}

type peerMessage struct {
	Origin string
	NickName string
	Content string
}

type clientMessage struct {
	Text string `json:"message"`
	To string `json:"to"`
}

type ChatMessage struct {
	Nickname string
	Text string
	Encrypted bool
}

type nickNameMessage struct {
	NickName string
}

func getMessages(w http.ResponseWriter, r *http.Request) {

	lock.Lock()

	var body, err = ioutil.ReadAll(r.Body)
	CheckError(err)

	body = bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))

	data, err := json.Marshal(updateMessage{server.MsgBuffer})

	CheckError(err)

	w.Header().Set("Content-Type", "application/json")

	_, err = w.Write(data)
	CheckError(err)

	server.MsgBuffer = make([]peerMessage, 0, 100)
	lock.Unlock()

}

func getPeers(w http.ResponseWriter, r *http.Request) {

	var body, err = ioutil.ReadAll(r.Body)
	CheckError(err)

	body = bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))

	var msg = nickNameMessage{}
	err = json.Unmarshal(body, &msg)
	CheckError(err)

	nickName = msg.NickName

	data, err := json.Marshal(setupMessage{server.Addr.IP.String(), server.Peers})

	CheckError(err)

	w.Header().Set("Content-Type", "application/json")

	_, err = w.Write(data)
	CheckError(err)

}

func getRoot(w http.ResponseWriter, r *http.Request) {

	encrypted = r.Header.Get("encryption_set") == "on"

	http.FileServer(http.Dir(".")).ServeHTTP(w, r)

}


func newMsg(w http.ResponseWriter, r *http.Request) {

	var body, err = ioutil.ReadAll(r.Body)
	CheckError(err)

	body = bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))

	var msg = clientMessage{}
	err = json.Unmarshal(body, &msg)
	CheckError(err)

	var pckt = ChatMessage{}

	text := ""

	if encrypted {
		text, err = EncryptString(msg.Text)
		CheckError(err)
	} else {
		text = msg.Text
	}

	pckt.Text = text
	pckt.Nickname = nickName
	pckt.Encrypted = encrypted

	var packetBytes, err4 = protobuf.Encode(&pckt)
	CheckError(err4)

	peerAddr, err := net.ResolveUDPAddr("udp4", msg.To+":5001")
	CheckError(err)

	var udpConn, err5 = net.DialUDP("udp4", nil, peerAddr)
	CheckError(err5)

	var _, err6 = udpConn.Write(packetBytes)
	CheckError(err6)

	fmt.Println("SENDING MSG : \""+text+"\" TO : "+msg.To)

	udpConn.Close()

}

func waitForMessages() {

	var connection, err= net.ListenUDP("udp4", &net.UDPAddr{server.Addr.IP, 5001, server.Addr.Zone})
	CheckError(err)

	defer connection.Close()

	var buffer = make([]byte, 2048)

	for {

		var sz, from, errRcv = connection.ReadFromUDP(buffer)
		CheckError(errRcv)

		if errRcv == nil {

			var pckt= &ChatMessage{}

			var errDecode = protobuf.Decode(buffer[:sz], pckt)
			CheckError(errDecode)

			if pckt.Text != "" {

				text := ""

				if pckt.Encrypted {
					text, err = DecryptString(pckt.Text)
					CheckError(err)
				} else {
					text = pckt.Text
				}

				fmt.Println("RCVD MDG : \"" + pckt.Text + "\" FROM :" + from.IP.String() + " ("+pckt.Nickname+")")
				lock.Lock()
				server.MsgBuffer = append(server.MsgBuffer,
					peerMessage{from.IP.String(), pckt.Nickname, text})
				lock.Unlock()
			}

		}

	}

}

func main() {

	server = &ServerInfo{}

	server.Peers = os.Args[2:]
	udpAddr, err := net.ResolveUDPAddr("udp4", os.Args[1]+":5000")
	CheckError(err)

	go waitForMessages()

	server.Addr = udpAddr

	r := mux.NewRouter()

	r.Methods("POST").Subrouter().HandleFunc("/newMessage", newMsg)//HandleFunc("/", newMsg)
	r.Methods("GET").Subrouter().HandleFunc("/getMessages", getMessages)//HandleFunc("/", newMsg)
	r.Methods("POST").Subrouter().HandleFunc("/getPeers", getPeers)//HandleFunc("/", newMsg)
	r.Methods("GET").Subrouter().HandleFunc("/", getRoot)//HandleFunc("/", newMsg)
	r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(http.Dir("."+"/assets/"))))
	//r.Handle("/", http.FileServer(http.Dir(".")))

	log.Println(http.ListenAndServe(":8080", r))

}
