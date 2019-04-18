package main

import (
	"errors"
	"fmt"
	"time"
)

const errorOccurrenceCount int = 5
const tooManyErrorsDelay time.Duration = 10 * time.Second //seconds

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
			// if oldest error and newest are within tooManyErrorsDelay * errorOccurrenceCount seconds then wait a bit longer
			if !errorOccurrences[0].IsZero() && errorOccurrences[errorOccurrenceCount-1].Sub(errorOccurrences[0]).Seconds() < tooManyErrorsDelay.Seconds()*float64(errorOccurrenceCount) {
				time.Sleep(tooManyErrorsDelay)
			} else {
				time.Sleep(time.Second)
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
