package network


// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
// TODO: gå over og tune alle tider (deadline, timeouts, etc)
// TODO: endre statemottak til å sende unmarshalled state via kanal til master
// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

import (
	 "./network/bcast"
	 "./network/localip"
	 // "../timer"
	 "../elevstate"

	"fmt"
	 "time"
	"log"
	"net"
	// "sync"
)

type Msgstruct struct{
	Message string
}

type slaveConn struct{
	conn *net.TCPConn
	lastalive time.Time
	port string
	killread chan bool
}

type OperatingMode int

const (
	OM_Master OperatingMode = iota
	OM_Slave
	OM_Independent
)

const stringLengthIP = 15
const stringLengthPort = 6

const bcInterval = 25
const cycleTime = 500
const conTime = 50
const stateDeadline = 100
const keepSlave = 250


const bcPort = 15660

//var mux sync.Mutex

func Network(port string, priority int, modeChan chan<- OperatingMode, sigSend <-chan bool, sigReceived chan<-[]byte){
	var conn *net.TCPConn
	mode := OM_Independent
	var IPorPort string

	myIP, err := localip.LocalIP()
	if err != nil {
		log.Println(err)
		myIP = "DISCONNECTED"
	}

	BCChan 	:= make(chan Msgstruct)
	LChan 	:= make(chan Msgstruct)

	go bcast.Transmitter(bcPort, BCChan)
	go bcast.Receiver(bcPort, LChan)

	for{

// Bare for å unngå problemer
		select{
		case <-sigSend:
			break
		default:
			break
		}

		switch(mode){
		case OM_Slave:
			masterIP := IPorPort
			mode = slaveNetwork(conn, port, masterIP, BCChan, LChan, sigSend, sigReceived)
			modeChan <-mode
			break

		case OM_Master:
			firstslaveport := IPorPort
			mode = masterNetwork(conn, firstslaveport, myIP, BCChan, LChan, sigSend, sigReceived)
			modeChan <-mode
			break

		default:
			conn, IPorPort, mode = independentNetwork(port, myIP, priority, BCChan, LChan, sigSend)
			modeChan <-mode
			break
		}
	}
}

func independentNetwork(port string, IP string, priority int, BCChan chan<- Msgstruct, LChan <-chan Msgstruct, sigSend <-chan bool) (*net.TCPConn, string, OperatingMode){

	ipmsg				:= make(chan string)
  BCkill		  := make(chan bool)
	listenkill	:= make(chan bool)

	go broadcast(port, BCChan, BCkill)
	go listenForMsg(nil, ipmsg, LChan, listenkill)


	for{
		for start := time.Now(); time.Since(start) < time.Duration(priority*cycleTime)*time.Millisecond; {
			select{
			case a := <-ipmsg:
				// fmt.Println("Attempting to become slave...")
				start = time.Now()
				conn, err := slaveConnect(a, port)
				if err != nil {
					log.Println(err)
					}else{
						BCkill <-true
						listenkill <-true
						return conn, a, OM_Slave
					}

			// Just in case
			case <-sigSend:
				break
			default:
				break
			}
		}
		// fmt.Println("Attempting to become master...")
		BCkill <-true
		listenkill <-true
		conn, slaveport := masterConnAttempt(IP, BCChan, LChan)
		if conn == nil {
			// fmt.Println("Master connection attempt failed!")
		}else{
			// fmt.Println("Master connection attempt succeeded!")
			return conn, slaveport, OM_Master
		}
		// fmt.Println("Resuming independent functions...")
		go broadcast(port, BCChan, BCkill)
		go listenForMsg(nil, ipmsg, LChan, listenkill)

	}

}

func masterNetwork(firstConn *net.TCPConn, firstport string, masterIP string, BCChan chan<- Msgstruct, LChan <-chan Msgstruct, sigSend <-chan bool, sigReceived chan<-[]byte) (OperatingMode){
	slavemap := make(map[string]slaveConn)

	var  firstslave slaveConn
	firstslave.conn = firstConn
	firstslave.lastalive = time.Now()
	firstslave.port = firstport
	firstslave.killread = make(chan bool)
	slavemap[firstport] = firstslave

	newslave := false

	// stateReceiver			:= make(chan []byte)
	slaveport 				:= make(chan string)
	BCkill 						:= make(chan bool)
	listenkill				:= make(chan bool)

	go broadcast(masterIP, BCChan, BCkill)
	go listenForMsg(slaveport, nil, LChan, listenkill)
	go listenForState(slavemap[firstport].conn, sigReceived, slavemap[firstport].killread)

	for start := time.Now(); time.Since(start) < time.Duration(cycleTime)*time.Millisecond;{
		select{
		case a:= <-slaveport:
			slavemap, newslave = slavemapHandler(slavemap, a)
			if newslave {
				go listenForState(slavemap[a].conn, sigReceived, slavemap[a].killread)
			}
			newslave = false

		case <-sigSend:
			for _, slave := range slavemap{
				go sendState(slave.conn, true)
			}

		// case a := <-stateReceiver:
		// 	go receiveState(a, true)
		// 	// Choose one of these
		// 	sigReceived <-true
		// 	// go func(){sigReceived <-true}()


		default:
			break

		}

		slavemap = slavemapCheckTime(slavemap)
		if len(slavemap) > 0 {
				start = time.Now()
		}

	}
	BCkill <-true
	listenkill <-true
	return OM_Independent
}

func slaveNetwork(conn *net.TCPConn, port string, masterIP string, BCChan chan<- Msgstruct, LChan <-chan Msgstruct, sigSend <-chan bool, sigReceived chan<-[]byte) (OperatingMode){
	// stateReceiver 	:= make(chan []byte, 2)
	ipmsg 					:= make(chan string)
	BCkill 					:= make(chan bool)
	listenkill 			:= make(chan bool)
	readkill				:= make(chan bool)

	go broadcast(port, BCChan, BCkill)
	go listenForMsg(nil, ipmsg, LChan, listenkill)
	go listenForState(conn, sigReceived, readkill)


	for start := time.Now(); time.Since(start) < time.Duration(cycleTime)*time.Millisecond;{
		select{
		case a := <-ipmsg:
			if a == masterIP {
				start = time.Now()
			}

		case <-sigSend:
			go sendState(conn, false)

		// case a := <-stateReceiver:
		// 	go receiveState(a, false)
		// 	// Choose one of these
		// 	sigReceived <-true
		// 	// go func(){sigReceived <-true}()

		default:
			break
		}
	}
	BCkill <-true
	listenkill <-true
	readkill <-true
	conn.Close()
	return OM_Independent
}

func broadcast(msg string, bcchan chan<- Msgstruct, kill <-chan bool){
	for{
		select{
		case <-kill:
			// fmt.Println("Broadcast has been killed")
			return
		case <-time.After(bcInterval*time.Millisecond):
			bcchan <- Msgstruct{msg}
		}
	}
}

func listenForMsg(portstr chan<- string, ipstr chan<- string, msgchan <-chan Msgstruct, kill <-chan bool){
	pending := false
	for{
		select{

		case <-kill:
			return

		case a := <-msgchan:

			switch(len(a.Message)){

			case stringLengthIP:
				if ipstr != nil && !pending {
					pending = true
					go func() {
						ipstr <-a.Message
						pending = false
					}()
				}
				break

			case stringLengthPort:
				if portstr != nil && !pending {
					pending = true
					go func() {
						portstr <-a.Message
						pending = false
					}()
				}
				break

			default:
				break
			}

		}
	}
}

func masterConnAttempt(masterIP string, BCChan chan<- Msgstruct, LChan <-chan Msgstruct) (*net.TCPConn, string){

	slavePort 	:= make(chan string)
	BCkill 			:= make(chan bool)
	listenkill	:= make(chan bool)

	go broadcast(masterIP, BCChan, BCkill)
	go listenForMsg(slavePort, nil, LChan, listenkill)

	for start := time.Now(); time.Since(start) < time.Duration(cycleTime)*time.Millisecond; {
		select{
		case a := <-slavePort:
			conn, err := masterTCPConnect(a)
			if err != nil {
				log.Println(err)
			}else{
				BCkill <-true
				listenkill <-true
				return conn, a
			}

		default:
			break
		}
	}
	BCkill <-true
	listenkill <-true
	return nil, ""
}

func masterTCPConnect(port string) (*net.TCPConn, error){
	listenAddr, err := net.ResolveTCPAddr("tcp", port)
  if err != nil {
    return nil, err
  }

  listenConn, err := net.ListenTCP("tcp", listenAddr)
  if err != nil {
    return nil, err
  }
  defer listenConn.Close()

	err = listenConn.SetDeadline(time.Now().Add(time.Duration(conTime)*time.Millisecond))
	if err != nil {
    return nil, err
  }

  conn, err := listenConn.AcceptTCP()
  if err != nil {
    return nil, err
  }

	msg := []byte("Connection accepted")
  _, err = conn.Write(msg)
  if err != nil {
    return nil, err
  }

  return conn, nil

}

func slaveConnect(masterIP string, port string) (*net.TCPConn, error){
	MAddr, err := net.ResolveTCPAddr("tcp", masterIP+port)
	if err != nil {
		return nil, err
	}

	// Initiate connection with master
	conn, err := net.DialTCP("tcp", nil, MAddr)
	if err != nil {
		return nil, err
	}

	buffer := make([]byte, 1024)

  // Check connection status
  err = conn.SetDeadline(time.Now().Add(time.Duration(conTime)*time.Millisecond))
  if err != nil {
    return nil, err
  }

  _, err = conn.Read(buffer)
  if err != nil {
    return nil, err
  }

  //reset deadline
  err = conn.SetDeadline(time.Time{})
  if err != nil {
    return nil, err
  }

  return conn, nil
}

func slavemapHandler(slavemap map[string]slaveConn, port string) (map[string]slaveConn, bool) {
	_, ok := slavemap[port]
	if ok{
		var slave slaveConn
		slave.conn = slavemap[port].conn
		slave.lastalive = time.Now()
		slave.port = slavemap[port].port
		slave.killread = slavemap[port].killread
		slavemap[port] = slave
	}else{
		conn, err := masterTCPConnect(port)
		if err != nil {
			fmt.Println("Error connecting to:", port)
			log.Println(err)
			return slavemap, false
		}
		var slave slaveConn
		slave.conn = conn
		slave.lastalive = time.Now()
		slave.port = port
		slave.killread = make(chan bool)
		slavemap[port] = slave
	}
	return slavemap, !ok
}

func slavemapCheckTime(slavemap map[string]slaveConn) (map[string]slaveConn){
	for port, slave := range slavemap{
		if time.Since(slave.lastalive) > time.Duration(keepSlave)*time.Millisecond {
			slave.killread <-true
			slave.conn.Close()
			delete(slavemap, port)
		}
	}
	return slavemap
}

func sendState(conn *net.TCPConn, isMaster bool){
	var toSend []byte
	var err error

	if isMaster {
		toSend, err = elevstate.RetrieveSystemStateBytes()
		if err != nil {
			log.Println(err)
			return
		}
	}else{
		toSend, err = elevstate.RetrieveStateBytes()
		if err != nil {
			log.Println(err)
			return
		}
	}

	_, err = conn.Write(toSend)
	if err != nil {
		log.Println(err)
	}
	return
}

func ReceiveState(state []byte, isMaster bool){

	if isMaster {
		// Master updates entire system state
		err := elevstate.SystemStateUpdate(state)
		if err != nil {
			log.Println(err)
		}
	}else{
		// Slave stores entire system state
		system := elevstate.UnmarshalSystem(state)
		err := elevstate.SystemStore(system)
		if err != nil {
			log.Println(err)
		}
	}
}

func listenForState(conn *net.TCPConn, state chan<- []byte, kill <-chan bool){
	buffer := make([]byte, 1024)
	pending := false

	for{
		select{
		case <-kill:
			return
		default:
			break
		}
		err := conn.SetReadDeadline(time.Now().Add(time.Duration(stateDeadline)*time.Second))
		if err != nil {
			log.Println(err)
		}

		n, err := conn.Read(buffer)
		if err == nil {
			if !pending{
				pending = true
				go func(){
					state <-buffer[0:n]
					pending = false
					}()
			}
		}

	}
}
