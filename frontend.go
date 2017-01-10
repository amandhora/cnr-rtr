package main

import (
	"fmt"
	"github.com/cinarra/cnr-rtr/gen/crp"
	"github.com/garyburd/redigo/redis"
	"github.com/golang/protobuf/proto"
	zmq "github.com/pebbe/zmq3"
	"log"
	"sync"
)

type FrontEndCtx struct {
	fSock *zmq.Socket
	bSock *zmq.Socket
	rConn redis.Conn
	wPool chan struct{}
	wg    sync.WaitGroup
}

func initFrontendContext(
	conf *Configuration,
	fsock *zmq.Socket,
	bsock *zmq.Socket,
	rconn redis.Conn) *FrontEndCtx {

	// Pool for workers
	pool := createDspWorkerPool(conf.DspWrkCnt)

	return &FrontEndCtx{
		fSock: fsock,
		bSock: bsock,
		rConn: rconn,
		wPool: pool,
	}
}

func createDspWorkerPool(nWorkers int) chan struct{} {

	// Create global pool for workers
	dspWrkPool := make(chan struct{}, nWorkers)

	return dspWrkPool
}

func (ctx *FrontEndCtx) processDspReq() {

	log.Println("FE: Processing Dsp Req")

	// Clear entry from pool on completion
	defer func() { <-ctx.wPool }()

	// Receive Req from DSP
	recv, _ := ctx.fSock.RecvBytes(0)
	counts.Add("DSP Recv Req", 1)

	// Message retrieved from the queue
	ctx.wg.Done()

	// Unmarshal request
	dspReq := getDspReq(recv)

	appuid := dspReq.GetUserId()

	// Lookup appuid in cache
	data, err := getCacheEntry(ctx.rConn, appuid)

	if err == nil {

		// Entry found in cache
		counts.Add("RTR Cache Hit", 1)
		log.Println("RTR: Cache HIT ", appuid)

		// Unmarshal byte data extracted from cache
		rsp := &crp.CrpMsg{}
		err = proto.Unmarshal(data, rsp)
		if err != nil {
			log.Fatal("Cache data marshaling error: ", err)
		}

		// Send recommendation to DSP
		dspRsp := createDspRsp(rsp.GetRecRepV2())
		ctx.fSock.SendBytes(dspRsp, 0)
		counts.Add("DSP Send Rec", 1)

	} else {

		// Not found in cache, Request CCS
		counts.Add("RTR Cache Miss", 1)
		log.Println("RTR: *** cache MISS ***  ", appuid)

		//  Create and send Rec Req to CCS
		recReq := createSimRecReq(appuid)
		ctx.bSock.SendBytes(recReq, 0)
		counts.Add("MGW Send", 1)

		// Send No Rec to DSP
		dspRsp := createDspRspNoRec(appuid)
		ctx.fSock.SendBytes(dspRsp, 0)
		counts.Add("DSP Send NoRec", 1)
	}

}

// Main routine to handle messages received from DSP
func startFrontEndLoop(conf *Configuration, rConn redis.Conn) {

	//  Socket to receive messages for DSP
	fSock, _ := zmq.NewSocket(zmq.DEALER)
	defer fSock.Close()
	fSock.Connect(fmt.Sprint("tcp://", conf.Dsp.Addr, ":", conf.Dsp.Port))

	//  Socket to send messages to backend
	bSock, _ := zmq.NewSocket(zmq.DEALER)
	defer bSock.Close()
	bSock.Connect(fmt.Sprint("tcp://", conf.MgwSend.Addr, ":", conf.MgwSend.Port))

	log.Println("FE: Starting frontend worker")
	ctx := initFrontendContext(conf, fSock, bSock, rConn)

	//  Register sockets
	poller := zmq.NewPoller()
	poller.Add(fSock, zmq.POLLIN)

	//  Process received messages
	for {
		sockets, _ := poller.Poll(-1)

		for _, socket := range sockets {
			switch s := socket.Socket; s {

			case fSock:
				// Set entry in the worker pool
				ctx.wPool <- struct{}{}

				ctx.wg.Add(1)

				// Start Goroutine to process Dsp Req
				go ctx.processDspReq()

				// Wait till message is read from queue
				ctx.wg.Wait()

			}
		}
	}

}
