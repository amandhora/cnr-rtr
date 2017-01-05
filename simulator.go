package main

import (
	"fmt"
	"github.com/cinarra/cnr-atca-repo/rtr/gen/crp"
	"github.com/golang/protobuf/proto"
	zmq "github.com/pebbe/zmq3"
	"log"
	"time"
)

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
