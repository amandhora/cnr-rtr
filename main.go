package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"
)

var (
	ConfigFile string = "rtr-conf.json"
	LogFile    string = "rtr.log"
)

func init() {
	http.HandleFunc("/rtr/stats", rtrStatsHandler)
	counts.Set("StartTime", time.Now())
}

func main() {

	fd, err := os.OpenFile(LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("error opening file: %v", err)
	}
	defer fd.Close()

	log.SetOutput(fd)
	log.Println("Logger Initialized")

	// Read config from json
	conf, err := readJsonConfig(ConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	// Connect to redis server
	//rConn, err := redis.Dial("tcp", ":10000")
	rConn, err := redis.Dial("tcp", conf.Redis.Addr+":"+strconv.Itoa(conf.Redis.Port))
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		http.ListenAndServe(":8080", http.DefaultServeMux)
	}()

	// Start Emulators
	startSimulators(&conf)

	// Start CLI Worker
	go startCliLoop(&conf)

	// Start Backend Worker
	go startBackEndLoop(&conf, rConn)

	// Start FrontEnd Worker
	go startFrontEndLoop(&conf, rConn)

	// Catch Ctrl + C
	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)

	go func() {
		for _ = range signalChan {
			log.Println("\nReceived an interrupt, stopping services...\n")
			//cleanup(services, c)
			cleanupDone <- true
		}
	}()

	// Block main routine till we receive signal
	<-cleanupDone
}

/*
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

type mgwRsp struct {
	recRsp     *crp.CrpMsg
	identifier string
}

type mgwTask struct {
	identifier string
	task       chan mgwRsp
	recReq     []byte
}

func procDspReq(dspSock *zmq.Socket, c redis.Conn, ch1 chan mgwTask) {

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

		dspRsp := createDspRsp(rsp.recRsp.GetRecRepV2())
		dspSock.SendBytes(dspRsp, 0)
		counts.Add("DSP Send Rec", 1)

	case rsp := <-ch3:

		dspRsp := createDspRsp(rsp.recRsp.GetRecRepV2())
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
*/
