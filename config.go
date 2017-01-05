package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type ConnectConf struct {
	Addr string
	Port int
}

/* Struct for storing config read from JSON file */
type Configuration struct {
	CliPort int
	Redis   ConnectConf
	Dsp     ConnectConf
	MgwSend ConnectConf
	MgwRecv ConnectConf
}

/* Global - To store RTR config */
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
