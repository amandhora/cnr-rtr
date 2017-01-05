package main

import (
	"github.com/cinarra/cnr-atca-repo/rtr/gen/crp"
	"github.com/golang/protobuf/proto"
	"log"
)

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
