package network

import (
	 "./network/bcast"
	 // "./network/localip"
	 "../timer"

	"fmt"
	 "time"
	"log"
	"net"
	"sync"
)

type Msgstruct struct{
	Message string
}

type slaveConn struct{
	conn *net.TCPConn
	lastalive time.Time
	port string
}

const stringLengthIP = 15
const stringLengthPort = 6

const bcintervalms = 1000

const bcPort = 15657

var mux sync.Mutex

/*
func main(){

	var port string
	flag.StringVar(&port, "port", ":12345", "port of this peer")
	flag.Parse()

	if port[0] != ':' {
		port = ":"+port
	}
	if len(port) > 6 {
		fmt.Println("Error: Invalid port")
		return
	}

	for{
    var test int
    fmt.Println("Choose test:")
    fmt.Println("1: Master")
    fmt.Println("2: Slave")

    fmt.Println("0: exit")

    _, err := fmt.Scan(&test)
    if err != nil {
      log.Fatal(err)
    }

    switch(test){
		case 0:
			fmt.Println("Exiting...")
			return

		case 1:
			masterBCTest()
			break

		case 2:
			slaveBCTest(port)
			break

		default:
			fmt.Println("Invalid input, try again")
			break

		}
	}
}

func masterBCTest(){

	localIP, err := localip.LocalIP()
	if err != nil {
		fmt.Println(err)
		localIP = "DISCONNECTED"
		fmt.Println("Network status: ", localIP)
		return
	}

	IPTx := make(chan Msgstruct)
	msg := make(chan Msgstruct)
	slaveport := make(chan string)

	go bcast.Transmitter(bcPort, IPTx)
	go bcast.Receiver(bcPort, msg)

	go listenForMsg(slaveport, nil, msg)

	go func() {
			for {
				IPTx <- Msgstruct{localIP}
				time.Sleep(1 * time.Second)
		}
	}()

	for{
		select{
		case a := <-slaveport:
			fmt.Println("Port received: ", a)
		}
	}

}

func slaveBCTest(port string){

	portTx := make(chan Msgstruct)
	msg := make(chan Msgstruct)
	masterIP := make(chan string)

	go bcast.Transmitter(bcPort, portTx)
	go bcast.Receiver(bcPort, msg)

	go listenForMsg(nil, masterIP, msg)

	go func() {
		for{
			portTx <- Msgstruct{port}
			time.Sleep(1*time.Second)
		}
	}()

	for{
		select{
		case a := <-masterIP:
			fmt.Println("IP received: ", a)
		}
	}

}
*/


/*
func main(){
	var port string
	var priority int

	flag.StringVar(&port, "port", ":12345", "port of this peer")
	flag.IntVar(&priority, "priority", 1, " master priority of this peer")
	flag.Parse()

	if port[0] != ':' {
		port = ":"+port
	}
	if len(port) > stringLengthPort {
		fmt.Println("Error: Invalid port")
		return
	}

	localIP, err := localip.LocalIP()
	if err != nil {
		fmt.Println(err)
		localIP = "DISCONNECTED"
		fmt.Println("IP: ", localIP)
		return
	}

	mode := make(chan string)

	go Network(port, localIP, priority, mode)

	for{
		select{
		case a := <-mode:
			fmt.Println("This elevator is now", a)
		}
	}

}
*/


func Network(port string, IP string, priority int, modeChan chan<- string){
	var conn *net.TCPConn
	var mode string
	var IPorPort string

	BCChan 	:= make(chan Msgstruct)
	LChan 	:= make(chan Msgstruct)

	go bcast.Transmitter(bcPort, BCChan)
	go bcast.Receiver(bcPort, LChan)

	for{
		switch(mode){
		case "slave":
			masterIP := IPorPort
			mode = slaveNetwork(conn, port, masterIP, BCChan, LChan)
			modeChan <-mode
			break

		case "master":
			firstslaveport := IPorPort
			mode = masterNetwork(conn, firstslaveport, IP, BCChan, LChan)
			modeChan <-mode
			break

		default:
			conn, IPorPort, mode = independentNetwork(port, IP, priority, BCChan, LChan)
			modeChan <-mode
			break
		}
	}
}


func independentNetwork(port string, IP string, priority int, BCChan chan<- Msgstruct, LChan <-chan Msgstruct) (*net.TCPConn, string, string){

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
		}
	}

}

func masterNetwork(firstConn *net.TCPConn, firstport string, masterIP string, BCChan chan<- Msgstruct, LChan <-chan Msgstruct) (string){
	slavemap := make(map[string]slaveConn)
	slavemap[firstport] = slaveConn{firstConn, time.Now(), firstport}

	slaveport 				:= make(chan string)
	BCkill 						:= make(chan bool)
	listenkill				:= make(chan bool)

	go broadcast(masterIP, BCChan, BCkill)
	go listenForMsg(slaveport, nil, LChan, listenkill)

	for start := time.Now(); time.Since(start) < 5*time.Second;{
		select{
		case a:= <-slaveport:
			fmt.Println("Port received:", a)
			slavemap = slavemapHandler(slavemap, a)

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

func slaveNetwork(conn *net.TCPConn, port string, masterIP string, BCChan chan<- Msgstruct, LChan <-chan Msgstruct) (string){
	ipmsg 			:= make(chan string)
	BCkill 			:= make(chan bool)
	listenkill 	:= make(chan bool)

	go broadcast(port, BCChan, BCkill)
	go listenForMsg(nil, ipmsg, LChan, listenkill)


	for start := time.Now(); time.Since(start) < 5*time.Second;{
		select{
		case a := <-ipmsg:
			if a == masterIP {
				start = time.Now()
			}

		default:
			break
		}
	}
	BCkill <-true
	listenkill <-true
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

	for{
		mux.Lock()
		mux.Unlock()
		select{

		case <-kill:
			return

		case a := <-msgchan:

			switch(len(a.Message)){

			case stringLengthIP:
				if ipstr != nil {
					ipstr <-a.Message
				}
				break

			case stringLengthPort:
				if portstr != nil {
					portstr <-a.Message
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
	listenkill	:= make(chan bool, 2)

	go broadcast(masterIP, BCChan, BCkill)
	go listenForMsg(slavePort, nil, LChan, listenkill)

	for start := time.Now(); time.Since(start) < 5*time.Second; {
		select{
		case a := <-slavePort:
			mux.Lock()
			fmt.Println("1")
			conn, err := masterTCPConnect(a)
			fmt.Println("2")
			if err != nil {
				log.Println(err)
				mux.Unlock()
			}else{
				BCkill <-true
				listenkill <-true
				mux.Unlock()
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

func slavemapHandler(slavemap map[string]slaveConn, port string) (map[string]slaveConn) {
	_, ok := slavemap[port]
	if ok{
		slavemap[port] = slaveConn{slavemap[port].conn, time.Now(), port}
	}else{
		conn, err := masterTCPConnect(port)
		if err != nil {
			fmt.Println("Error connecting to:", port)
			log.Println(err)
			return slavemap
		}
		slavemap[port] = slaveConn{conn, time.Now(), port}
	}
	return slavemap
}

func slavemapCheckTime(slavemap map[string]slaveConn) (map[string]slaveConn){
	for port, slave := range slavemap{
		if time.Since(slave.lastalive) > 5*time.Second {
			delete(slavemap, port)
		}
	}
	return slavemap
}

// Master slave connection tests
/*
func main(){
	fmt.Println("Master-slave connection test")
	localip, err := localip.LocalIP()
	if err != nil {
		log.Println(err)
		return
	}

	go func(){

		conn, err := slaveConnect(localip, ":12345")
		if err != nil {
			log.Println(err)
			return
		}

		// buffer := make([]byte, 1024)

		for start := time.Now(); time.Since(start) < 30*time.Second;{

				// err := conn.SetReadDeadline(time.Now().Add(1*time.Second))
				// if err != nil {
				// 	log.Println(err)
				// 	return
				// }
				//
				// n, err := conn.Read(buffer)
				// if err != nil {
				// 	log.Println(err)
				// }
				// fmt.Println("Received: ", string(buffer[0:n]))


				_, err = conn.Write([]byte("hello there"))
				if err != nil {
					log.Println(err)
					return
				}
				time.Sleep(1*time.Second)
		}
	}()

	conn, err := masterTCPConnect(":12345")
	if err != nil {
		log.Println(err)
		return
	}
	buffer := make([]byte, 1024)
	for start := time.Now(); time.Since(start) < 30*time.Second;{

		// _, err = conn.Write([]byte("hello there"))
		// if err != nil {
		// 	log.Println(err)
		// 	return
		// }
		// time.Sleep(1*time.Second)


		err := conn.SetReadDeadline(time.Now().Add(1*time.Second))
		if err != nil {
			log.Println(err)
			return
		}

		n, err := conn.Read(buffer)
		if err != nil {
			log.Println(err)
		}
		fmt.Println("Received: ", string(buffer[0:n]))
	}
}


func main() {
	go slave()
	master()
}

func slave(){
	BCChan := make(chan Msgstruct)
	LChan  := make(chan Msgstruct)

	go bcast.Transmitter(bcPort, BCChan)
	go bcast.Receiver(bcPort, LChan)

	go func(){
		for{
			BCChan <- Msgstruct{":12345"}
			time.Sleep(1*time.Second)
		}
	}()

	var conn *net.TCPConn
	var err error


	for (conn == nil) {
		select{
		case a := <-LChan:
			if len(a.Message) == stringLengthIP {
				conn, err = slaveConnect(a.Message, ":12345")
				if err != nil {
					log.Println(err)
				}
			}
		}
	}

	buffer := make([]byte, 1024)
	for{
		err = conn.SetReadDeadline(time.Now().Add(2*time.Second))
		if err != nil {
			log.Println(err)
		}

		n, err := conn.Read(buffer)
		if err != nil {
			log.Println(err)
		}
		fmt.Println("Slave received: ", string(buffer[0:n]), "from master")
	}
}

func master(){
	localip, err := localip.LocalIP()
	if err != nil {
		log.Println(err)
		localip = "192.168.180.127"
	}

	BCChan := make(chan Msgstruct)
	LChan  := make(chan Msgstruct)

	go bcast.Transmitter(bcPort, BCChan)
	go bcast.Receiver(bcPort, LChan)

	conn,_ := masterConnAttempt(localip, BCChan, LChan)


	for{

		_, err := conn.Write([]byte("Hello there!"))
		if err != nil {
			log.Println(err)
		}
		time.Sleep(1*time.Second)
	}
}
*/
