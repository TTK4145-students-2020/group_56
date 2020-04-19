package network


// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
// TODO: gå over og tune alle tider (deadline, timeouts, etc)
// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

import (
	 "./network/bcast"
	 "./network/localip"
	 "../timer"
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

const stringLengthIP = 15
const stringLengthPort = 6

const bcintervalms = 1000

const bcPort = 15657

//var mux sync.Mutex

func Network(port string, priority int, modeChan chan<- string, sigSend <-chan bool){
	var conn *net.TCPConn
	var mode string
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
		case "slave":
			masterIP := IPorPort
			mode = slaveNetwork(conn, port, masterIP, BCChan, LChan, sigSend)
			modeChan <-mode
			break

		case "master":
			firstslaveport := IPorPort
			mode = masterNetwork(conn, firstslaveport, myIP, BCChan, LChan, sigSend)
			modeChan <-mode
			break

		default:
			conn, IPorPort, mode = independentNetwork(port, myIP, priority, BCChan, LChan, sigSend)
			modeChan <-mode
			break
		}
	}
}


func independentNetwork(port string, IP string, priority int, BCChan chan<- Msgstruct, LChan <-chan Msgstruct, sigSend <-chan bool) (*net.TCPConn, string, string){

	ipmsg				:= make(chan string)
	timerreset  := make(chan bool)
  timeout     := make(chan bool)
  BCkill		  := make(chan bool)
	listenkill	:= make(chan bool)

	go broadcast(port, BCChan, BCkill)
	go listenForMsg(nil, ipmsg, LChan, listenkill)
	go timer.NetworkTimer(timerreset, timeout, float64(priority*5))

	for{
		select{
		case <-timeout:
			fmt.Println("Attempting to become master...")
			BCkill <-true
			listenkill <-true
			conn, slaveport := masterConnAttempt(IP, BCChan, LChan)
			if conn == nil {
				fmt.Println("Master connection attempt failed!")
			}else{
				fmt.Println("Master connection attempt succeeded!")
				return conn, slaveport, "master"
			}
			fmt.Println("Resuming independent functions...")
			go broadcast(port, BCChan, BCkill)
			go listenForMsg(nil, ipmsg, LChan, listenkill)
			go timer.NetworkTimer(timerreset, timeout, float64(20+priority*5))

		case a := <-ipmsg:
			fmt.Println("Attempting to become slave...")
			timerreset <-true
			conn, err := slaveConnect(a, port)
			if err != nil {
				log.Println(err)
			}else{
				BCkill <-true
				listenkill <-true
				timerreset <-false
				return conn, a, "slave"
			}

		// Just in case
		case <-sigSend:
			break
		}
	}

}

func masterNetwork(firstConn *net.TCPConn, firstport string, masterIP string, BCChan chan<- Msgstruct, LChan <-chan Msgstruct, sigSend <-chan bool) (string){
	slavemap := make(map[string]slaveConn)

	var  firstslave slaveConn
	firstslave.conn = firstConn
	firstslave.lastalive = time.Now()
	firstslave.port = firstport
	firstslave.killread = make(chan bool)
	slavemap[firstport] = firstslave

	newslave := false

	stateReceiver			:= make(chan []byte)
	slaveport 				:= make(chan string)
	BCkill 						:= make(chan bool)
	listenkill				:= make(chan bool)

	go broadcast(masterIP, BCChan, BCkill)
	go listenForMsg(slaveport, nil, LChan, listenkill)
	go listenForState(slavemap[firstport].conn, stateReceiver, slavemap[firstport].killread)

	for start := time.Now(); time.Since(start) < 5*time.Second;{
		select{
		case a:= <-slaveport:
			fmt.Println("Port received:", a)
			slavemap, newslave = slavemapHandler(slavemap, a)
			if newslave {
				go listenForState(slavemap[a].conn, stateReceiver, slavemap[a].killread)
			}
			newslave = false

		case <-sigSend:
			for _, slave := range slavemap{
				go sendState(slave.conn, true)
			}

		case a := <-stateReceiver:
			go receiveState(a, true)


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
	return "independent"
}

func slaveNetwork(conn *net.TCPConn, port string, masterIP string, BCChan chan<- Msgstruct, LChan <-chan Msgstruct, sigSend <-chan bool) (string){
	stateReceiver 	:= make(chan []byte, 2)
	ipmsg 					:= make(chan string)
	BCkill 					:= make(chan bool)
	listenkill 			:= make(chan bool)
	readkill				:= make(chan bool)

	go broadcast(port, BCChan, BCkill)
	go listenForMsg(nil, ipmsg, LChan, listenkill)
	go listenForState(conn, stateReceiver, readkill)


	for start := time.Now(); time.Since(start) < 5*time.Second;{
		select{
		case a := <-ipmsg:
			if a == masterIP {
				start = time.Now()
			}

		case <-sigSend:
			go sendState(conn, false)

		case a := <-stateReceiver:
			go receiveState(a, false)

		default:
			break
		}
	}
	BCkill <-true
	listenkill <-true
	readkill <-true
	conn.Close()
	return "independent"
}

func broadcast(msg string, bcchan chan<- Msgstruct, kill <-chan bool){
	for{
		select{
		case <-kill:
			// fmt.Println("Broadcast has been killed")
			return
		case <-time.After(bcintervalms*time.Millisecond):
			bcchan <- Msgstruct{msg}
		}
	}
}

func listenForMsg(portstr chan<- string, ipstr chan<- string, msgchan <-chan Msgstruct, kill <-chan bool){
	pending := false
	for{
		// mux.Lock()
		// mux.Unlock()
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

	for start := time.Now(); time.Since(start) < 5*time.Second; {
		select{
		case a := <-slavePort:
			// mux.Lock()
			fmt.Println("1")
			conn, err := masterTCPConnect(a)
			fmt.Println("2")
			if err != nil {
				log.Println(err)
				// mux.Unlock()
			}else{
				BCkill <-true
				listenkill <-true
				// mux.Unlock()
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

	err = listenConn.SetDeadline(time.Now().Add(1000*time.Millisecond))
	if err != nil {
    return nil, err
  }

  conn, err := listenConn.AcceptTCP()
  if err != nil {
    return nil, err
  }

	msg := []byte("Connection accepted")
  //fmt.Println("Connection accepted")
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
  err = conn.SetDeadline(time.Now().Add(250*time.Millisecond))
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
		if time.Since(slave.lastalive) > 5*time.Second {
			fmt.Println("I kill")
			slave.killread <-true
			fmt.Println("I killed")
			slave.conn.Close()
			delete(slavemap, port)
		}
	}
	return slavemap
}

// To do: legg til mutex, så ikke statefiler skaper problemer (gjør i elevstate)
// både sendState() og receiveState() bør kjøres som goroutiner,
// så ikke mutex blokker resten av nettverksmodulen
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

func receiveState(state []byte, isMaster bool){

	if isMaster {
		// Master updates entire system state
		err := elevstate.SystemStateUpdate(state)
		if err != nil {
			log.Println(err)
		}
	}else{
		// Slave stores entire system state
		err := elevstate.SystemStore(state)
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
		err := conn.SetReadDeadline(time.Now().Add(1*time.Second))
		if err != nil {
			fmt.Println("I'm troublesome")
			log.Println(err)
		}

		n, err := conn.Read(buffer)
		if err != nil {
			// log.Println(err)
		}else{
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
