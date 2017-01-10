package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type ConnectConf struct {
	Addr string
	Port int
}

/* Struct for storing config read from JSON file */
type Configuration struct {
	SimStart  bool
	DspWrkCnt int
	CcsWrkCnt int
	CliPort   int
	Redis     ConnectConf
	Dsp       ConnectConf
	MgwSend   ConnectConf
	MgwRecv   ConnectConf
}

func readJsonConfig(configfile string) (Configuration, error) {

	/* To store RTR config */
	var conf Configuration

	file, err := os.Open(configfile)
	if err != nil {
		log.Fatal("Config file is missing: ", configfile)
	}

	decoder := json.NewDecoder(file)

	err = decoder.Decode(&conf)
	if err != nil {
		fmt.Println("Failed to Read Config:", err)
	} else {
		fmt.Println("*********************************")
		fmt.Printf("CONFIG:\n  %+v\n", conf)
		fmt.Println("*********************************")
	}

	return conf, err
}
