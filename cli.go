package main

import (
	"bytes"
	"expvar"
	"fmt"
	"log"
	"regexp"
)

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
