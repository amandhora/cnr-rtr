package main

import (
	"fmt"
	"github.com/cinarra/cnr-atca-repo/rtr/gen/crp"
	"github.com/garyburd/redigo/redis"
	"github.com/golang/protobuf/proto"
	zmq "github.com/pebbe/zmq3"
	"log"
	"net/http"
	"time"
)

func init() {
	http.HandleFunc("/rtr/stats", rtrStatsHandler)
	counts.Set("StartTime", time.Now())

}

func mgwReceiver(ch chan mgwRsp) {

	//  Socket to receive messages for MGW
	mgwReceiver, _ := zmq.NewSocket(zmq.DEALER)
	defer mgwReceiver.Close()
	mgwReceiver.Connect(fmt.Sprint("tcp://", conf.MgwRecv.Addr, ":", conf.MgwRecv.Port))

	//  Process messages from MGW Emulators
	poller := zmq.NewPoller()
	poller.Add(mgwReceiver, zmq.POLLIN)

	for {
		sockets, _ := poller.Poll(-1)
		for _, socket := range sockets {
			switch s := socket.Socket; s {

			case mgwReceiver:
				msg, _ := s.RecvBytes(0)

				counts.Add("MGW Recv", 1)

				// Un pack REC rsp
				resp := getRecRsp(msg)
				appuid := resp.RecRep.GetAppUids().GetUserId()

				task := mgwRsp{
					recRsp:     resp,
					identifier: appuid,
				}
				go func() {
					ch <- task
				}()
			}
		}
	}
}

func mgwDispatcher(ch1 chan mgwTask, ch2 chan mgwRsp) {

	//  Socket to send messages to MGW
	mgwSender, _ := zmq.NewSocket(zmq.DEALER)
	defer mgwSender.Close()
	mgwSender.Connect(fmt.Sprint("tcp://", conf.MgwSend.Addr, ":", conf.MgwSend.Port))

	m := make(map[string]chan mgwRsp)

	for {
		select {
		case req := <-ch1:
			// Process REC REQ from DSP

			//fmt.Println("DISPATCHER: sending mgw")
			m[req.identifier] = req.task

			//  Send results to MGW
			mgwSender.SendBytes(req.recReq, 0)
			counts.Add("MGW Send", 1)

		case rsp := <-ch2:
			// Process REC RSP from MGW
			//fmt.Println("DISPATCHER: rcvd mgw")

			ch3 := m[rsp.identifier]
			ch3 <- rsp
		}
	}
}

func startMgwProxy() chan mgwTask {

	ch1 := make(chan mgwTask, 10)
	ch2 := make(chan mgwRsp, 10)

	go mgwReceiver(ch2)

	go mgwDispatcher(ch1, ch2)

	return ch1
}

func main() {
	// Read config from json
	err := readJsonConfig()
	if err != nil {
		panic(err)
	}

	c, err := redis.Dial("tcp", ":10000")
	if err != nil {
		log.Print(err)
	}

	go func() {
		http.ListenAndServe(":8080", http.DefaultServeMux)
	}()

	//  Socket to receive messages for DSP
	dspSock, _ := zmq.NewSocket(zmq.DEALER)
	defer dspSock.Close()
	dspSock.Connect(fmt.Sprint("tcp://", conf.Dsp.Addr, ":", conf.Dsp.Port))

	//  Socket to send messages to MGW
	mgwSender, _ := zmq.NewSocket(zmq.DEALER)
	defer mgwSender.Close()
	mgwSender.Connect(fmt.Sprint("tcp://", conf.MgwSend.Addr, ":", conf.MgwSend.Port))

	//  Socket for CLI input
	re, err := initCliExp()
	cliSock, _ := zmq.NewSocket(zmq.ROUTER)
	defer cliSock.Close()
	cliSock.Bind(fmt.Sprint("tcp://*:", conf.CliPort))

	// Start Emulators
	ch := startMgwProxy()
	go dspEmulator()
	go mgwEmulator()
	go cliEmulator()

	//  Process messages from all Emulators
	poller := zmq.NewPoller()
	poller.Add(dspSock, zmq.POLLIN)
	//poller.Add(mgwReceiver, zmq.POLLIN)
	poller.Add(cliSock, zmq.POLLIN)

	//  Process messages from all sockets
	for {
		sockets, _ := poller.Poll(-1)
		for _, socket := range sockets {
			switch s := socket.Socket; s {
			case dspSock:

				// Received request from DSP
				handleDspReq(s, c, ch)

			case cliSock:
				identity, _ := cliSock.Recv(0)
				cmd, _ := cliSock.Recv(0)
				rsp := executeCliCmd(re, cmd)
				cliSock.Send(identity, zmq.SNDMORE)
				cliSock.Send(rsp, 0)

			}
		}
	}
	fmt.Println()
}

type mgwRsp struct {
	recRsp     *crp.CrpMsg
	identifier string
}

type mgwTask struct {
	identifier string
	task       chan mgwRsp
	recReq     []byte
}

func handleDspReq(dspSock *zmq.Socket, c redis.Conn, ch1 chan mgwTask) {

	ch2 := make(chan mgwRsp, 1)
	ch3 := make(chan mgwRsp, 1)

	// Receive Req from DSP
	r, _ := dspSock.RecvBytes(0)
	dspReq := getDspReq(r)
	counts.Add("DSP Recv Req", 1)

	appuid := dspReq.GetUserId()

	go func() {

		data, err := getCacheEntry(c, appuid)
		if err == nil {

			// Entry found in cache
			counts.Add("RTR Cache Hit", 1)
			log.Println("RTR: Cache HIT ")

			// Unmarshal byte data extracted from cache
			rsp := &crp.CrpMsg{}
			err = proto.Unmarshal(data, rsp)
			if err != nil {
				log.Fatal("Cache data marshaling error: ", err)
			}

			// Send respons back right away
			task1 := mgwRsp{
				identifier: appuid,
				recRsp:     rsp,
			}
			ch2 <- task1

		} else {

			// Not found in cache, Request CCS
			counts.Add("RTR Cache Miss", 1)
			log.Println("RTR: *** cache MISS ***")
			//  Do the work

			recReq := createSimRecReq(appuid)

			//  Send results to MGW
			task2 := mgwTask{
				identifier: appuid,
				task:       ch3,
				recReq:     recReq,
			}
			go func() {
				ch1 <- task2
			}()

		}

	}()

	select {
	case rsp := <-ch2:

		dspRsp := createDspRsp(rsp.recRsp.GetRecRep())
		dspSock.SendBytes(dspRsp, 0)
		counts.Add("DSP Send Rec", 1)

	case rsp := <-ch3:

		dspRsp := createDspRsp(rsp.recRsp.GetRecRep())
		dspSock.SendBytes(dspRsp, 0)
		counts.Add("DSP Send Rec", 1)

		// Cache MGW response
		data, err := proto.Marshal(rsp.recRsp)
		if err != nil {
			log.Fatal("marshaling error: ", err)
		}

		err = setCacheEntry(c, rsp.identifier, data)
		if err != nil {
			log.Println("RTR: fail to store in cache: ", rsp.identifier)
		} else {
			log.Println("RTR: stored in cache: ", rsp.identifier)
		}

	case <-time.After(10 * time.Millisecond):

		log.Println("Life is to short to wait that long. Expire request after 10 Msec")
		dspRsp := createDspRspNoRec(appuid)
		dspSock.SendBytes(dspRsp, 0)

		counts.Add("DSP Send NoRec", 1)
	}

}
