package main

import (
	"fmt"
	"github.com/cinarra/cnr-rtr/gen/crp"
	"github.com/golang/protobuf/proto"
	zmq "github.com/pebbe/zmq3"
	"log"
	"time"
)

/* Start Simulators for testing */
func startSimulators(conf *Configuration) {

	/* If Config is configured to start simulators */
	if conf.SimStart {

		go dspEmulator(conf)
		go mgwEmulator(conf)
		go cliEmulator(conf)
	}

}

// Send msg every second
func dspEmulator(conf *Configuration) {
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
func mgwEmulator(conf *Configuration) {

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
				appuid := req.RecReqV2.GetAppUids().GetUserId()
				log.Println("MGW: Sending REP ", appuid)

				r := createSimRecRsp(appuid)
				//  Send results to sink
				sender.SendBytes(r, 0)

			} else {
				log.Println("MGW: Lets DROP ")
			}
		}

		toggle++
	}
}

func cliEmulator(conf *Configuration) {

	//  Socket to receive messages on
	sock, _ := zmq.NewSocket(zmq.DEALER)
	defer sock.Close()
	sock.Connect(fmt.Sprint("tcp://localhost:", conf.CliPort))

	cmd1 := "show rtr stats"
	cmd2 := "show rtr config"
	//  Process tasks forever
	for {

		//  Periodic CLI cmd generator
		time.Sleep(20 * time.Second)

		// Send to RTR
		sock.Send(cmd1, 0)

		// Read from RTR
		resp, _ := sock.Recv(0)
		fmt.Println("*********************************")
		fmt.Println("CLI CMD: ", cmd1)
		fmt.Println("CLI RSP: ", resp)
		fmt.Println("*********************************")

		// Send to RTR
		sock.Send(cmd2, 0)

		// Read from RTR
		resp, _ = sock.Recv(0)
		fmt.Println("*********************************")
		fmt.Println("CLI CMD: ", cmd2)
		fmt.Println("CLI RSP: ", resp)
		fmt.Println("*********************************")
	}
}
