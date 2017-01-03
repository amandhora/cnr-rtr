package main

import (
	"bytes"
	"encoding/json"
	"expvar"
	"fmt"
	"github.com/cinarra/cnr-atca-repo/rtr/gen/crp"
	"github.com/garyburd/redigo/redis"
	"github.com/golang/protobuf/proto"
	zmq "github.com/pebbe/zmq3"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
)

var (
	counts = expvar.NewMap("counters")
)

// REST handle to export local metrics
func rtrStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "{\n")
	first := true
	counts.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")
}

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

func mgwEmulator() {
	//  Socket to receive messages on
	receiver, _ := zmq.NewSocket(zmq.DEALER)
	defer receiver.Close()
	receiver.Bind(fmt.Sprint("tcp://*:", conf.MgwSend.Port))

	//  Socket to send messages to
	sender, _ := zmq.NewSocket(zmq.DEALER)
	defer sender.Close()
	sender.Bind(fmt.Sprint("tcp://*:", conf.MgwRecv.Port))

	toggle := 0

	//  Process tasks forever
	for {
		s, err := receiver.RecvBytes(0)
		if err != nil {
			log.Println(err)
		} else {

			req := &crp.CrpMsg{}
			err := proto.Unmarshal(s, req)
			if err != nil {
				log.Fatal("unmarshaling error: ", err)
			}

			//fmt.Printf("MGW: REQ: %+v", req)

			if toggle%2 == 0 {
				log.Println("MGW: Sending REP ")

				r := createRecRsp(req.RecReq.GetAppUids().GetUserId())
				//  Send results to sink
				sender.SendBytes(r, 0)

			} else {
				log.Println("MGW: Lets DROP ")
			}
		}

		toggle++
	}
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

// Send msg every second
func dspEmulator() {
	//  Socket to receive messages on
	sock, _ := zmq.NewSocket(zmq.DEALER)
	defer sock.Close()
	sock.Bind(fmt.Sprint("tcp://*:", conf.Dsp.Port))

	//  Process tasks forever
	for {

		identity := "TheAppuid"

		time.Sleep(3 * time.Second)
		//  Work generator
		log.Println()
		log.Println("DSP: Sending Req for User [", identity, "]")

		req := createDspReq(identity)
		// Send to RTR
		sock.SendBytes(req, 0)

		// Read from RTR
		s, _ := sock.RecvBytes(0)
		rsp := getDspRsp(s)

		if rsp.GetNoadsCodes() != 0 {
			log.Println("DSP: NO-REC for [", rsp.GetUserId(), "]")
		} else {
			log.Println("DSP: Recv Recommendation for [", rsp.GetUserId(), "]")
			log.Println("   : ", rsp.GetCampItemId())
		}
	}
}

func createRecReq(s string) []byte {

	recRequest := &crp.CrpRecReq{
		Adxid: proto.Uint32(2),
		AppUids: &crp.CnrAppUIDs{
			UserId: proto.String(s),
		},
		TransId: &crp.CnrUTID{
			Cpid: proto.Uint64(1000),
			Tid:  proto.Uint64(2000),
		},
		Debug: proto.Uint32(99),
		Flags: proto.Uint32(0),
	}

	req := &crp.CrpMsg{
		Version: proto.Uint32(0x00000001),
		MsgType: proto.Uint32(uint32(crp.CrpMsgType_CRP_MSG_RECREQ)),
		Flags:   proto.Uint32(0x00000000),
		RecReq:  recRequest,
	}

	//fmt.Printf("RSP: %+v", req)

	data, err := proto.Marshal(req)
	if err != nil {
		log.Fatal("marshaling error: ", err)
	}

	return data
}

func createRecRsp(s string) []byte {

	recResponse := &crp.CrpRecRep{
		AppUids: &crp.CnrAppUIDs{
			UserId: proto.String(s),
		},
		TransId: &crp.CnrUTID{
			Cpid: proto.Uint64(1000),
			Tid:  proto.Uint64(2000),
		},
		Ads: []*crp.CtpTgtAds{
			&crp.CtpTgtAds{
				AdId:  proto.Uint64(50000000001),
				ObjId: proto.Uint64(10),
			},
			&crp.CtpTgtAds{
				AdId:  proto.Uint64(50000000002),
				ObjId: proto.Uint64(10),
			},
		},
		TypeAd: proto.Uint32(99),
		Flags:  proto.Uint32(0),
	}

	rsp := &crp.CrpMsg{
		Version: proto.Uint32(0x00000001),
		MsgType: proto.Uint32(uint32(crp.CrpMsgType_CRP_MSG_RECREP)),
		Flags:   proto.Uint32(0x00000000),
		RecRep:  recResponse,
	}

	//fmt.Printf("RSP: %+v", rsp)

	data, err := proto.Marshal(rsp)
	if err != nil {
		log.Fatal("rec rsp marshaling error: ", err)
	}

	return data
}

func getRecRsp(data []byte) *crp.CrpMsg {

	recResp := &crp.CrpMsg{}
	err := proto.Unmarshal(data, recResp)
	if err != nil {
		log.Fatal("rec rsp unmarshaling error: ", err)
	}

	//log.Printf("Unmarshalled to: %+v", recResp)

	return recResp
}

func setCacheEntry(c redis.Conn, app string, value []byte) error {

	_, err := c.Do("SET", app, value, "EX", 10)
	if err != nil {
		log.Print(err)
	}

	return err
}

func getCacheEntry(c redis.Conn, app string) ([]byte, error) {

	//data, err := redis.String(c.Do("GET", app))
	data, err := redis.Bytes(c.Do("GET", app))

	//fmt.Println("REDIS: ", app)

	return data, err
}

func initCliExp() (*regexp.Regexp, error) {
	re, err := regexp.Compile("^show rtr stats")
	if err != nil {
		log.Fatal(err)
	}

	//fmt.Println(re.FindStringSubmatch("set rtr stats"))
	//fmt.Println(re.FindStringSubmatch("lets show rtr stats"))
	return re, err
}

func executeCliCmd(re *regexp.Regexp, cmd string) string {

	//fmt.Println("CLI: ", cmd)
	rsp := "Not Found"

	exec := re.FindStringSubmatch(cmd)
	if len(exec) != 0 {
		buf := new(bytes.Buffer)
		counts.Do(func(kv expvar.KeyValue) {
			fmt.Fprintf(buf, "\n%q: %s", kv.Key, kv.Value)
		})
		rsp = buf.String()
	}

	return rsp
}

func cliEmulator() {

	//  Socket to receive messages on
	sock, _ := zmq.NewSocket(zmq.DEALER)
	defer sock.Close()
	sock.Connect(fmt.Sprint("tcp://localhost:", conf.CliPort))

	cmd := "show rtr stats"
	//  Process tasks forever
	for {

		//  Periodic CLI cmd generator
		time.Sleep(20 * time.Second)

		// Send to RTR
		sock.Send(cmd, 0)

		// Read from RTR
		resp, _ := sock.Recv(0)
		fmt.Println("*********************************")
		fmt.Println("CLI CMD: ", cmd)
		fmt.Println("CLI RSP: ", resp)
		fmt.Println("*********************************")
	}
}

type ConnectConf struct {
	Addr string
	Port int
}

type Configuration struct {
	CliPort int
	Redis   ConnectConf
	Dsp     ConnectConf
	MgwSend ConnectConf
	MgwRecv ConnectConf
}

var conf Configuration

func readJsonConfig() error {
	file, _ := os.Open("rtr_conf.json")
	decoder := json.NewDecoder(file)

	err := decoder.Decode(&conf)
	if err != nil {
		fmt.Println("Failed to Read Config:", err)
	} else {
		fmt.Println("*********************************")
		fmt.Printf("CONFIG: %v\n", conf)
		fmt.Println("*********************************")
	}

	return err
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

			recReq := createRecReq(appuid)

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

func createDspReq(s string) []byte {

	req := &crp.DspReq{
		DspId:   proto.Uint32(1),
		TransId: proto.Uint64(0x000000001000),
		UserId:  proto.String(s),
	}

	//fmt.Printf("RSP: %+v", req)

	data, err := proto.Marshal(req)
	if err != nil {
		log.Fatal("Dsp req marshaling error: ", err)
	}

	return data
}

func getDspReq(req []byte) *crp.DspReq {

	//fmt.Printf("RSP: %+v", req)

	data := &crp.DspReq{}

	err := proto.Unmarshal(req, data)
	if err != nil {
		log.Fatal("unmarshaling error: ", err)
	}

	return data
}

func createDspRspNoRec(s string) []byte {

	rsp := &crp.DspRsp{
		UserId:     proto.String(s),
		NoadsCodes: proto.Uint64(1),
	}

	//fmt.Printf("RSP: %+v", rsp)

	data, err := proto.Marshal(rsp)
	if err != nil {
		log.Fatal("unmarshaling error: ", err)
	}

	return data
}

func createDspRsp(r *crp.CrpRecRep) []byte {

	rsp := &crp.DspRsp{
		UserId: proto.String(r.GetAppUids().GetUserId()),
	}

	// Populate campaign Item list
	c := r.GetAds()
	rsp.CampItemId = make([]uint64, len(c))

	for i, cid := range c {
		rsp.CampItemId[i] = cid.GetAdId()
	}

	//fmt.Printf("RSP: %+v", rsp)

	data, err := proto.Marshal(rsp)
	if err != nil {
		log.Fatal("marshaling error: ", err)
	}

	return data
}

func getDspRsp(rsp []byte) *crp.DspRsp {

	//fmt.Printf("RSP: %+v", rsp)

	data := &crp.DspRsp{}

	err := proto.Unmarshal(rsp, data)
	if err != nil {
		log.Fatal("unmarshaling error: ", err)
	}

	return data
}
