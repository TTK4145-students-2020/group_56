package network

import (
  "fmt"
  "net"
  "log"
  "time"
  "math/rand"
)

type TCPConnection struct{
  Conn        *net.TCPConn
  Port        string
  Active      bool
}

func MasterIPBroadcast(msg string, port string){
  addr := "255.255.255.255" + port

	BCAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.DialUDP("udp", nil, BCAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	for {

		_, err := conn.Write([]byte(msg))
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(1 * time.Second)

	}
}

func SlaveIPRead(port string, IP chan<- string, connection chan<- bool){
  BCAddr, err := net.ResolveUDPAddr("udp",port)
  if err != nil {
		log.Println(err)
		//return
	}

  conn, err := net.ListenUDP("udp", BCAddr)
  if err != nil {
		log.Println(err)
		//return
	}
  defer conn.Close()

  err = conn.SetDeadline(time.Now().Add(3 * time.Second)) //find appropriate timeout time
  if err != nil {
		log.Println(err)
		//return
	}

  buffer := make([]byte, 1024)
  for{
    n, err := conn.Read(buffer)
    if err != nil {
      log.Println(err)
      connection <- false
      return
    }
    fmt.Println(string(buffer[0:n]))
    IP <- string(buffer[0:n])

    time.Sleep(1*time.Second)
  }
}

func SlaveTCPConnect(IP string, port string) (conn *net.TCPConn, err error){

  MAddr, err := net.ResolveTCPAddr("tcp", IP+port)
  if err != nil {
    log.Println(err)
    return
  }

  // Initiate connection with master
  conn, err = net.DialTCP("tcp", nil, MAddr)
  if err != nil {
    log.Println(err)
    return
  }

  buffer := make([]byte, 1024)

  // Check connection status
  _, err = conn.Read(buffer)
  if err != nil {
    log.Println(err)
    return
  }

  return
}

func MasterTCPConnect(port string) (conn *net.TCPConn, err error){
  listenAddr, err := net.ResolveTCPAddr("tcp", port)
  if err != nil {
    log.Println(err)
    return
  }

  listenConn, err := net.ListenTCP("tcp", listenAddr)
  if err != nil {
    log.Println(err)
    return
  }
  defer listenConn.Close()

  conn, err = listenConn.AcceptTCP()
  if err != nil {
    log.Println(err)
    return
  }

  msg := []byte("Connection accepted")
  fmt.Println("Connection accepted")
  _, err = conn.Write(msg)
  if err != nil {
    log.Println(err)
    return
  }

  return

}

func MasterPortListen(listenPort string, port chan<- string, lerr chan<- error){

  listenAddr, err := net.ResolveUDPAddr("udp", listenPort)
  if err != nil {
    log.Println(err)
    lerr <- err
    return
  }

  conn, err := net.ListenUDP("udp", listenAddr)
  if err != nil {
    log.Println(err)
    lerr <- err
    return
  }
  defer conn.Close()

  buffer := make([]byte, 1024)
  for{
    n, err := conn.Read(buffer)
    if err != nil {
      log.Println(err)
      lerr <- err
      //return
    }
    fmt.Println(string(buffer[0:n]))
    port <- string(buffer[0:n])
  }
}

func SlaveBCPort(port string, BCPort string){
  rand.Seed(int64(time.Now().Nanosecond()))
  addr := "255.255.255.255" + BCPort

	BCAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Println(err)
	}

	conn, err := net.DialUDP("udp", nil, BCAddr)
	if err != nil {
		log.Println(err)
	}
	defer conn.Close()

	for {
		_, err := conn.Write([]byte(port))
    fmt.Println("broadcasting port...")
		if err != nil {
			log.Println(err)
		}
		time.Sleep(time.Duration(rand.Intn(3000)) * time.Millisecond)
	}
}

func MasterConnectionManager(MasterIP string, BCPort string, ListenPort string, conn chan<- TCPConnection) {
  lerr := make(chan error)
  slavePort := make(chan string)

  fmt.Println("hei")

  go MasterIPBroadcast(MasterIP, BCPort)
  go MasterPortListen(ListenPort, slavePort, lerr)

  for{
    select{
    case a := <-lerr:
      log.Println(a)
      go MasterPortListen(ListenPort, slavePort, lerr)

    case a := <- slavePort:
      connection, err := MasterTCPConnect(a)
      if err != nil {
  			log.Println(err)
  		}
      connectionstruct := TCPConnection{
        Conn: connection,
        Port: a,
        Active: true,
      }
      conn <-connectionstruct
    }
  }
}

func SlaveConnectionManager(Port string, BCPort string, ListenPort string, conn chan<- *net.TCPConn, connected chan<- bool){
    MasterIP := make(chan string)
    ConnectStat := make(chan bool)
    connstat := false

    go SlaveIPRead(ListenPort, MasterIP, ConnectStat)
    go SlaveBCPort(Port, BCPort)

    for{
      select{
      case a := <-MasterIP:
        fmt.Println("Master: "+a)
        if(!connstat){
          connect, err := SlaveTCPConnect(a, Port)
          if err != nil {
        		log.Println(err)
            connstat = false
            connected <- connstat
        		//return
        	}else{
            conn <- connect
            connstat = true
            connected <- connstat
          }
        }
      case a := <-ConnectStat:
        if(!a){
          go SlaveIPRead(ListenPort, MasterIP, ConnectStat)
        }
        //connstat = a
        //connected <- a
      }
    }

}

func ConnSetDeadline(conn net.TCPConn){
  err := conn.SetDeadline(time.Now().Add(3 * time.Second)) // Set appropriate SetDeadline
  if err != nil {
    log.Println(err)
    return
  }
}
