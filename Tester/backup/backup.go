package main

import (
	"flag"
	"time"

	"../../backup"
)

/*
	filen må bygges(?) (go build) før den testes (ikke go run)
*/

var isBackup = flag.Bool("backup", false, "Starts process as backup.")
var backupPort = flag.Int("bcport", 0, "Specifies what port the backup should dial.")

func main() {
	flag.Parse()
	if *isBackup {
		backup.AwaitPrimary(*backupPort)
	}
	backup.Launch()
	for {
		time.Sleep(5 * time.Second)
	}
}
