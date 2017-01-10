package main

import (
	"github.com/cinarra/cnr-rtr/gen/crp"
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

func createDspRsp(r *crp.CrpRecRepV2) []byte {

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

func getRecRsp(data []byte) *crp.CrpMsg {

	recResp := &crp.CrpMsg{}
	err := proto.Unmarshal(data, recResp)
	if err != nil {
		log.Fatal("rec rsp unmarshaling error: ", err)
	}

	//log.Printf("Unmarshalled to: %+v", recResp)

	return recResp
}

func createSimRecReq(s string) []byte {

	recRequest := &crp.CrpRecReqV2{
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
		Version:  proto.Uint32(0x00000001),
		MsgType:  proto.Uint32(uint32(crp.CrpMsgType_CRP_MSG_RECREQ)),
		Flags:    proto.Uint32(0x00000000),
		RecReqV2: recRequest,
	}

	//fmt.Printf("RSP: %+v", req)

	data, err := proto.Marshal(req)
	if err != nil {
		log.Fatal("marshaling error: ", err)
	}

	return data
}

func createSimRecRsp(s string) []byte {

	recResponse := &crp.CrpRecRepV2{
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
		Version:  proto.Uint32(0x00000001),
		MsgType:  proto.Uint32(uint32(crp.CrpMsgType_CRP_MSG_RECREP)),
		Flags:    proto.Uint32(0x00000000),
		RecRepV2: recResponse,
	}

	//fmt.Printf("RSP: %+v", rsp)

	data, err := proto.Marshal(rsp)
	if err != nil {
		log.Fatal("rec rsp marshaling error: ", err)
	}

	return data
}
