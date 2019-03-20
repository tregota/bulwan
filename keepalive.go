package main

import (
	"errors"
	"fmt"
	"time"
)

var errorOccurrenceCount = 5
var tooManyErrorsDelay = 10 * time.Second //seconds

// KeepAliveFunc funcs kan be kept alive using KeepAlive
type KeepAliveFunc func() (err error)

// KeepAlive keeps the provided KeepAliveFunc func alive
func KeepAlive(runThis KeepAliveFunc) (err error) {
	errorOccurrences := make([]time.Time, errorOccurrenceCount)

	for {
		err := panicCatcher(runThis)
		if err != nil {
			errorOccurrences = append(errorOccurrences[1:], time.Now())
			fmt.Printf("Error: %s\n", err)
			// too many errors too fast?
			if !errorOccurrences[errorOccurrenceCount-2].IsZero() && errorOccurrences[errorOccurrenceCount-1].Sub(errorOccurrences[errorOccurrenceCount-2]) < tooManyErrorsDelay+time.Second {
				if !errorOccurrences[0].IsZero() && errorOccurrences[errorOccurrenceCount-1].Sub(errorOccurrences[0]).Seconds() < float64(errorOccurrenceCount) {
					// all tracked errors within abount a second each length of time
					fmt.Printf("Too many errors, waiting %d seconds..\n", int(tooManyErrorsDelay.Seconds()))
					time.Sleep(tooManyErrorsDelay)
				} else {
					// two errors within tooManyErrorsDelay (+ a bit) period, wait 1 second
					time.Sleep(time.Second)
				}
			}
		}
	}
}

func panicCatcher(runThis KeepAliveFunc) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered:", r)
			// find out exactly what the error was and set err
			switch x := r.(type) {
			case error:
				err = x
			case string:
				err = errors.New(x)
			default:
				err = errors.New("Unknown panic")
			}
		}
	}()
	return runThis()
}
