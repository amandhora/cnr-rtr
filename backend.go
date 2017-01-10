package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/golang/protobuf/proto"
	zmq "github.com/pebbe/zmq3"
	"log"
	"sync"
)

type BackEndCtx struct {
	bSock *zmq.Socket
	rConn redis.Conn
	wPool chan struct{}
	wg    sync.WaitGroup
}

func initBackendContext(
	conf *Configuration,
	bsock *zmq.Socket,
	rconn redis.Conn) *BackEndCtx {

	// Pool for workers
	pool := createCcsWorkerPool(conf.CcsWrkCnt)

	return &BackEndCtx{
		bSock: bsock,
		rConn: rconn,
		wPool: pool,
	}
}

func createCcsWorkerPool(nWorkers int) chan struct{} {

	// Create global pool for workers
	ccsWrkPool := make(chan struct{}, nWorkers)

	return ccsWrkPool
}

func (ctx *BackEndCtx) processRecRsp() {

	log.Println("BE: Processing REC RSP")

	// Clear entry from pool on completion
	defer func() { <-ctx.wPool }()

	// Receive Rec Rsp from CCS
	msg, _ := ctx.bSock.RecvBytes(0)
	counts.Add("MGW Recv", 1)

	// Message retrieved from the queue
	ctx.wg.Done()

	// Un pack REC rsp
	rsp := getRecRsp(msg)
	appuid := rsp.RecRepV2.GetAppUids().GetUserId()

	// Cache MGW response
	data, err := proto.Marshal(rsp)
	if err != nil {
		log.Fatal("marshaling error: ", err)
	}

	/* Save REC RSP in cache */
	err = setCacheEntry(ctx.rConn, appuid, data)
	if err != nil {
		log.Println("RTR: fail to store in cache: ", appuid)
	} else {
		log.Println("RTR: stored in cache: ", appuid)
	}

}

// Main routine to handle messages received from CCS
func startBackEndLoop(conf *Configuration, rConn redis.Conn) {

	//  Socket to receive messages for MGW
	bSock, _ := zmq.NewSocket(zmq.DEALER)
	defer bSock.Close()
	bSock.Connect(fmt.Sprint("tcp://", conf.MgwRecv.Addr, ":", conf.MgwRecv.Port))

	log.Println("BE: Starting backend worker")
	ctx := initBackendContext(conf, bSock, rConn)

	//  Register sockets
	poller := zmq.NewPoller()
	poller.Add(bSock, zmq.POLLIN)

	//  Process received messages
	for {
		sockets, _ := poller.Poll(-1)

		for _, socket := range sockets {
			switch s := socket.Socket; s {

			case bSock:
				// Set entry in the worker pool
				ctx.wPool <- struct{}{}

				ctx.wg.Add(1)

				// Start Goroutine to process Rec Rsp from CCS
				go ctx.processRecRsp()

				// Wait till message is read from queue
				ctx.wg.Wait()
			}
		}
	}

}
