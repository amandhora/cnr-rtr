package main

import (
	"bytes"
	"expvar"
	"fmt"
	zmq "github.com/pebbe/zmq3"
	"regexp"
)

type command struct {
	re *regexp.Regexp

	// Handler routines slice
	handler func(*Configuration, string) string
}

/* Directory for storing all CLI commands */
type CommandRegister struct {

	// RTR configuration
	conf *Configuration

	// Slice for storing all supported commands
	cmds []*command
}

func (reg *CommandRegister) AddCommand(re string, handler func(*Configuration, string) string) {

	newcmd := &command{regexp.MustCompile(re), handler}

	// Append to the slice
	reg.cmds = append(reg.cmds, newcmd)
}

/* register CLI commands with corresponding handlers */
func (reg *CommandRegister) RegisterCMDHandlers() {

	reg.AddCommand("^show rtr stats", cliStatsHandler)
	reg.AddCommand("^show rtr config", cliConfigHandler)

}

/* CLI cmd handler */
func cliStatsHandler(conf *Configuration, cmd string) string {

	buf := new(bytes.Buffer)
	counts.Do(func(kv expvar.KeyValue) {
		fmt.Fprintf(buf, "\n%q: %s", kv.Key, kv.Value)
	})
	rsp := buf.String()

	return rsp
}

/* CLI cmd handler */
func cliConfigHandler(conf *Configuration, cmd string) string {

	rsp := fmt.Sprintf("%+v", *conf)

	return rsp
}

/* Execute the CLI command */
func (reg *CommandRegister) executeCliCmd(cmd string) string {

	var rsp string

	for _, command := range reg.cmds {
		matches := command.re.FindStringSubmatch(cmd)

		if matches != nil {

			//fmt.Println("matched uri: ", matches)
			rsp = command.handler(reg.conf, cmd)
			break
		}
	}

	return rsp
}

/* Initialize CLI handlers */
func initCli(conf *Configuration) *CommandRegister {

	regHandler := &CommandRegister{}

	regHandler.RegisterCMDHandlers()

	regHandler.conf = conf

	return regHandler

}

/*
func executeCliCmd1(h *CommandRegister, cmd string) string {

	//fmt.Println("CLI: ", cmd)
	rsp := "Not Found"

	rsp = h.executeCliCmd(cmd)

	return rsp
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

func executeCliCmdOld(re *regexp.Regexp, cmd string) string {

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
*/

// Main routine to handle messages received from cnr-mon
func startCliLoop(conf *Configuration) {

	handle := initCli(conf)

	cliSock, _ := zmq.NewSocket(zmq.ROUTER)
	defer cliSock.Close()
	cliSock.Bind(fmt.Sprint("tcp://*:", conf.CliPort))

	//  Register sockets
	poller := zmq.NewPoller()
	poller.Add(cliSock, zmq.POLLIN)

	//  Process received messages
	for {
		sockets, _ := poller.Poll(-1)

		for _, socket := range sockets {
			switch s := socket.Socket; s {

			case cliSock:

				identity, _ := cliSock.Recv(0)
				cmd, _ := cliSock.Recv(0)
				rsp := handle.executeCliCmd(cmd)
				cliSock.Send(identity, zmq.SNDMORE)
				cliSock.Send(rsp, 0)

			}
		}
	}

}
