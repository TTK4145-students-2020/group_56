package main

// TODO: endre navn fra requests til orders i elevator/elevator.go
// TODO: muligens endre navn på elevstate.go til noe minnegreier
// TODO: Motor power loss håndtering
// TODO: legge til orders for alle etasjelys i independent
// TODO: legge til active/inactive parameter i State
//       ... og ta høyde for den i findBestElev
//       ... og reassigne ordre fra en tapt slave

import (
	 "fmt"
	"time"
	//"net"

		"./elevio"
		"./fsm"
	//	"./timer"

	"flag"

	"./backup"
	"./chaffeur"
	"./network"
	"./elevstate"
	"./order_handler"
	// "os"
	// "io/ioutil"
	"log"
	// "strconv"
)

var (
	isBackup    = flag.Bool("backup", false, "Starts process as backup.")
	primaryPort = flag.Int("cbport", 0, "The port backup should dial to reach primary.")
  priority = flag.Int("priority", 1, "The priority when deciding master.")
  port = flag.String("port", ":0", "The port used when slave.")
	SimPort = flag.String("SimPort", ":15657", "The port used for the simulator")
	// numFloors = flag.Int("n", 4, "The number of floors.")
)

/*
type OperatingMode int

const (
	OM_Master operatingMode = iota
	OM_Slave
	OM_Independent
)
*/

func main() {
	flag.Parse()
	if *isBackup {
		backup.AwaitPrimary(*primaryPort)
	}
	// backup.Launch()

	// Initialisere state, enten fra fil eller fra scratch
  elevstate.StateInit(*port)

	// Lage kanaler for events som main må ta seg av
	event_switchMode := make(chan network.OperatingMode)
	event_localRequest := make(chan elevio.ButtonEvent)
  event_localStateChange := make(chan struct{}) // Senere: Heller chan elevator.Elevator, og så lagre state fra main ?
  event_networkMessage := make(chan []byte)

	// Request er ikke delegert, må delegeres av master
	// Order er delegert av master, må betjenes av heisen

	drv_order := make(chan elevio.ButtonEvent)
	drv_hallLights := make(chan [4][2]bool)

  net_sendState := make(chan bool) // Senere: chan struct{}

	// Initialisere nettverk og kjøring av heis
	go chaffeur.Chaffeur(event_localRequest, event_localStateChange, drv_order, drv_hallLights, *SimPort) // ??
  go network.Network(*port, *priority, event_switchMode, net_sendState, event_networkMessage)
	// Eller, blokkerende init-funksjoner som selv starter goroutiner ?
	// For å sikre at modulene er klare før vi begynner på for-select loopen

	// Anta mode independent
	mode := network.OM_Independent

	// For select hvor vi tar det som det kommer
	for {
		select {
		case e := <-event_switchMode:
			mode = e

      switch(mode){
      case network.OM_Master:
        err := elevstate.SystemInit()
        if err != nil {
          log.Println(err)
          break
        }
         // Hvis independent må hallLights oppdateres
      case network.OM_Independent:
        fsm.IndependentLights()
         // Legge inn hallCalls i Requests basert på HallLights

			 default:
				 break
      }

		case e := <-event_localRequest:
			// Hvis cab-call, delegere til seg selv og sende state til network
      if e.Button == elevio.BT_Cab { // Senere: Skal vi sende til main hvis det er cab?
        drv_order <- e
        break
      }
			// Hvis hall-call, switch mode
			switch mode {
			case network.OM_Independent:
				// gi seg selv ordre
        fsm.IndependentLights()
        drv_order <- e
			case network.OM_Master:
				// delegere ordre
        toMaster, hallLights, err := order_handler.AssignNewOrder(*port, e)
        if err != nil {
          log.Println(err)
          break
        }

        if toMaster {
          drv_order <- e
        }
        drv_hallLights <- hallLights
        net_sendState <- true
			case network.OM_Slave:
				// sende til master for delegering
        err := order_handler.NewRequest(e)
        if err != nil {
          log.Println(err)
          break
        }

        net_sendState <- true
      default:
        break
			}

		case <-event_localStateChange:
			switch mode {
			case network.OM_Independent:
        fsm.IndependentLights()
			case network.OM_Master:
				// Sende til slaver
        statebytes, err := elevstate.RetrieveStateBytes()
        if err != nil {
          log.Println(err)
          break
        }
        elevstate.SystemStateUpdate(statebytes)
        net_sendState <- true
			case network.OM_Slave:
				net_sendState <- true
      default:
        break
			}

      case e := <-event_networkMessage:
        switch mode {
        case network.OM_Independent:
          fmt.Println("WTF IS GOING ON!?!")
          break
        case network.OM_Master:
          // Antar state fra slave
          slaveState, err := elevstate.StateFromBytes(e)

          if err != nil {
            log.Println(err)
            break
          }

          unassignedRequests, _, err := order_handler.HandleStateFromSlave(slaveState)
					if err != nil {
						log.Println(err)
					}

					var toMaster bool
					var hallLights[4][2]bool

          for _, req := range unassignedRequests {
            toMaster, hallLights, err = order_handler.AssignNewOrder(*port, req)
						if err != nil {
							log.Println(err)
						}
            if toMaster {
              drv_order <- req
            }
          }
          drv_hallLights <- hallLights
          net_sendState <- true

        case network.OM_Slave:
          // Antar SystemState fra master
          systemState := elevstate.UnmarshalSystem(e)

          err := elevstate.SystemStore(systemState)

          if err != nil {
            log.Println(err)
            break
          }
					fmt.Println(systemState)
          newOrders, hallLights, err:= order_handler.HandleSystemStateFromMaster(systemState)
					if err != nil {
						log.Println(err)
					}
          drv_hallLights <- hallLights

          for _, order := range newOrders {
            drv_order <- order
          }


        default:
          break
        }
		}
		time.Sleep(10*time.Millisecond)
	}
}
