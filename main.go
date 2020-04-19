package main

import (
	 "fmt"
	"time"
	//"net"

	//	"./elevio"
	//	"./fsm"
	//	"./timer"

	"flag"

	"./backup"
	"./chaffeur"
	"./network"
	"./elevstate"
	// "os"
	// "io/ioutil"
	"log"
	// "strconv"
	"flag"
)

var (
	isBackup    = flag.Bool("backup", false, "Starts process as backup.")
	primaryPort = flag.Int("cbport", 0, "The port backup should dial to reach primary.")
	// numFloors = flag.Int("n", 4, "The number of floors.")
)

type operatingMode int

const (
	OM_Master operatingMode = iota
	OM_Slave
	OM_Independent
)

func main() {
	flag.Parse()
	if *isBackup {
		backup.AwaitPrimary(*primaryPort)
	}
	backup.Launch()

	// Initialisere state, enten fra fil eller fra scratch

	// Lage kanaler for events som main må ta seg av
	event_switchMode := make(chan operatingMode)
	event_request := make(chan elevio.ButtonEvent)
	event_order := make(chan order_handler.Order)
	event_stateChange := make(chan struct{})

	// Request er ikke delegert, må delegeres av master
	// Order er delegert av master, må betjenes av heisen

	drv_order := make(chan elevio.ButtonEvent)
	drv_hallLights := make(chan [4][2]bool)

	// Initialisere nettverk og kjøring av heis
	go chaffeur.Chaffeur(event_localRequest, event_stateChange, drv_order, drv_hallLights) // ??
	go network.Network(event_switchMode)                                                   // ??

	// Eller, blokkerende init-funksjoner som selv starter goroutiner ?
	// For å sikre at modulene er klare før vi begynner på for-select loopen

	// Anta mode independent
	mode := OM_Independent

	// For select hvor vi tar det som det kommer
	for {
		select {
		case e := <-event_switchMode:
			mode = e
		case e := <-event_localRequest:
			// Hvis cab-call, delegere til seg selv og sende state til network
			// Hvis hall-call, switch mode
			switch mode {
			case OM_Independent:
				// gi seg selv ordre
			case OM_Master:
				// delegere ordre
			case OM_Slave:
				// sende til master for delegering
			}
		case e := <-event_stateChange:
			switch mode {
			case OM_Independent:
				// ingenting?
			case OM_Master:
				// Sende til slaver
			case OM_Slave:
				// Sende til master
			}
		}
	}
}

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
} */

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
	var modestr string

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
	stateReceived := make(chan bool)

	err := elevstate.StateInit(port)
	if err != nil {
		return
	}

	go network.Network(port, priority, mode, send, stateReceived)
	for{
		select{
		case modestr = <-mode:
			fmt.Println("This elevator is now", modestr)
			if modestr == "master" {
				err := elevstate.SystemInit()
				if err != nil {
					log.Println(err)
				}
			}

		case <-time.After(1*time.Second):
			if (modestr == "slave" || modestr == "master"){
				go func(){send <-true}()
			}

		case <-stateReceived:
			fmt.Println("State Received")
		}
	}
}
