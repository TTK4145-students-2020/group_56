package timer

import (
	"time"
)

var endTime time.Time
var timerActive bool = false

const _pollRate = 20 * time.Millisecond

func getTime() (now time.Time) {
	now = time.Now()
	return
}

func Start(duration int) {
	endTime = getTime().Add(time.Second * 3)
	timerActive = true
	return
}

func Stop() {
	timerActive = false
	return
}
func timedOut() (timeout bool) {
	timeout = (timerActive && getTime().After(endTime))
	return
}

func PollTimer(receiver chan<- bool) {
	prev := false
	for {
		time.Sleep(_pollRate)
		v := timedOut()
		if v != prev {
			receiver <- v
		}
		prev = v
	}
}
