// NOTE: Need to communicate intentional shutdown to backup?

package backup

import (
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"
)

/* HOW TO USE:

var isBackup = flag.Bool("backup", false, "Starts process as backup.")
var backupPort = flag.Int("bcport", 0, "Specifies what port the backup should dial.")

func main() {
	flag.Parse()
	if (*isBackup) {
		backup.AwaitPrimary(*backupPort)
	}
	backup.Launch()
	...
*/

// Bør ikke ha slike globale verdier, samle de i en config-fil / objekt ?

var sendInterval = 1 * time.Second
var timeoutLimit = 3 * sendInterval

// Launch setter opp en UDP connection, og starter en backup prossess med riktig callbackPort
// Deretter setter den i gang en goroutine manageBackup som kommuniserer med backupen
func Launch() {
	Addr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.ListenUDP("udp", Addr)
	if err != nil {
		log.Fatal(err)
	}

	port := conn.LocalAddr().(*net.UDPAddr).Port

	execPath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	backupCommand := exec.Command(execPath, "-backup", "-bcport", strconv.Itoa(port))
	err = backupCommand.Start()
	if err != nil {
		log.Fatal(err)
	}

	go manageBackup(conn)
}

// AwaitPrimary kjøres som vanlig (blokkerende) funksjon fra main (når den startes som backup)
// Venter på at primary skal falle ut før den returnerer
func AwaitPrimary(port int) {
	primaryAddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(port))
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.DialUDP("udp", nil, primaryAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	for {
		_, err = conn.Write([]byte("1"))
		if err != nil {
			log.Fatal(err)
		}

		conn.SetReadDeadline(time.Now().Add(timeoutLimit))
		buffer := make([]byte, 1024)
		_, err := conn.Read(buffer)

		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				// This is a timeoutLimit error, meaning primary has missed the deadline
				break
			} else {
				log.Fatal(err)
			}
		}
		// Hvis vi skal gjøre noe med meldingene, f.eks hvis en type melding skal bety shutdown, kan det gjøres her
	}
}

func manageBackup(conn *net.UDPConn) {
	_, backupAddr, err := conn.ReadFrom(make([]byte, 1024))
	if err != nil {
		log.Fatal(err)
	}

	for {
		_, err := conn.WriteTo([]byte("1"), backupAddr)
		if err != nil {
			log.Fatal(err)
		}

		conn.SetReadDeadline(time.Now().Add(timeoutLimit))
		buffer := make([]byte, 1024)
		_, err = conn.Read(buffer)

		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				// This is a timeoutLimit error, meaning backup has missed the deadline
				break
			} else {
				log.Fatal(err)

			}
		}
		time.Sleep(sendInterval)
	}
	// If we get here, it means backup has failed, and we need to start a new one
	conn.Close()
	Launch()
}
