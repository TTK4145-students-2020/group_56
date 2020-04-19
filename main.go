package main

import (
	 "fmt"
	"time"
	//"net"

//	"./elevio"
//	"./fsm"
//	"./timer"

	"./network"
	"./elevstate"
	// "os"
	// "io/ioutil"
	"log"
	// "strconv"
	"flag"
)

/*func main() {
	fmt.Println("Started!")

	numFloors := 4

	elevio.Init("localhost:15657", numFloors)

	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)
	drv_timeout := make(chan bool)

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)
	go timer.PollTimer(drv_timeout)

	fsm.OnInitBetweenFloors()

	for {
		select {

		case a := <-drv_buttons:
			fsm.OnRequestButtonPress(a.Floor, a.Button)

		case a := <-drv_floors:
			fsm.OnFloorArrival(a)

		case a := <-drv_timeout:
			if a {
				fsm.OnDoorTimeout()
				timer.Stop()
			}
		}

		fsm.TransmitState()

		time.Sleep(20 * time.Millisecond)
	}
}*/



// master test
/*
func main(){
	MasterIP := "192.168.180.127"
	BCPort := ":12345"
	listenport := ":12346"
	//numslaves := 0
	var slaves []network.TCPConnection

	conns := make(chan network.TCPConnection)

	go network.MasterConnectionManager(MasterIP, BCPort, listenport, conns)

	for {
		select{
		case a := <-conns:
			//slaves[numslaves] = *a
			//slaves = append(slaves, a)
			addslave := true
			for i := 0; i< len(slaves);i++ {
				if(slaves[i].Port == a.Port){
					addslave = false
				}
			}
			if(addslave){
				slaves = append(slaves, a)
			}
		case <-time.After(1*time.Second):
			if(len(slaves)>0){
				jsonFile, err := os.Open("state.json")
			  if err != nil {
			      log.Println(err)
						fmt.Println(err)
			  }

				statebytes, err := ioutil.ReadAll(jsonFile)
			  if err != nil {
			      log.Println(err)
			  }

				for i := 0; i<len(slaves); i++ {
					if slaves[i].Active {
							fmt.Println("writing to slave num. "+strconv.Itoa(i))
							_, err = slaves[i].Conn.Write([]byte(statebytes))
							if err != nil {
								slaves[i].Active = false
								log.Println(err)
								fmt.Println(err)
							}
					}
				}
			}
		}

		jsonFile, err := os.Open("state.json")
	  if err != nil {
	      log.Println(err)
				fmt.Println(err)
	  }

		statebytes, err := ioutil.ReadAll(jsonFile)
	  if err != nil {
	      log.Println(err)
	  }

		for i := 0; i<len(slaves); i++ {
			fmt.Println("writing to slave num. "+strconv.Itoa(i))
			_, err = slaves[i].Conn.Write([]byte(statebytes))
			if err != nil {
		      log.Println(err)
					fmt.Println(err)
		  }
		}
	}
}




// Slave test

func main(){
	//MyIP := "192.168.180.127"
	MyPort := ":12347"
	listenport := ":12345"
	BCPort := ":12346"
//	var MasterIP string
	var conn net.TCPConn
	connstat := true
	reading := false

	connected := make(chan bool)
	connection := make(chan *net.TCPConn)
	readdata := make(chan []byte)
	readError := make(chan error)

	go network.SlaveConnectionManager(MyPort, BCPort, listenport, connection, connected)

	for{
		select{
		case a := <- connection:
			fmt.Println("Connected to Master")
			conn = *a

		case a := <-connected:
			fmt.Println("Connection status updated to ", strconv.FormatBool(a))
			connstat = a

		case a := <-readdata:
			fmt.Println("Received: ", string(a))
			err := ioutil.WriteFile("state.json", a, 0644)
		  if err != nil {
		      log.Println(err)
		  }

		case a := <-readError:
			fmt.Println("Error while reading")
			log.Println(a)
			go readData(conn, readdata, readError)

		}

		if(connstat && !reading){
			go readData(conn, readdata, readError)
			reading = true
		}

	}
}

func readData(conn net.TCPConn, data chan<- []byte, readError chan<- error){
	for{
		buffer := make([]byte, 1024)
		network.ConnSetDeadline(conn)
		n, err := conn.Read(buffer)
		if err != nil {
			log.Println(err)
			readError <- err
			return
		}else{
			data <- buffer[0:n]
		}
	}
}
*/

func main(){
	var port string
	var priority int

	flag.StringVar(&port, "port", ":12345", "This elevator's unique port")
	flag.IntVar(&priority, "priority", 1, "This elevator's master priority")
	flag.Parse()

	if port[0] != ':' {
		port = ":"+port
	}
	if len(port) > 6 {
		log.Println("Error: Invalid port")
		return
	}

	mode := make(chan string)
	send := make(chan bool)

	err := elevstate.StateInit(port)
	if err != nil {
		return
	}

	go network.Network(port, priority, mode, send)
	for{
		select{
		case a := <-mode:
			fmt.Println("This elevator is now", a)
			if a == "master" {
				err := elevstate.SystemInit()
				if err != nil {
					log.Println(err)
				}
			}

		case <-time.After(1*time.Second):
			send <-true
		}
	}
}
