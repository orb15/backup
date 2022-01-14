package main

import (
	"fmt"
	"time"
)

const (
	highestReasonableExponentThatWontOverflowInt = 46340
)

//convert a duration to a reasonably-looking string
func prettyTime(delta time.Duration) string {
	delta = delta.Round(time.Second)
	return delta.String()
}

//return a time.Duration that represents 2^(exponent) seconds
func calcBackoff(exponent int) (time.Duration, error) {

	//safety & sanity
	if exponent < 0 || exponent > highestReasonableExponentThatWontOverflowInt {
		return 0, fmt.Errorf("unsupported exponent value: %d", exponent)
	}
	if exponent == 0 {
		return 1, nil
	}

	//derive an int that is 2^(exponent). Golang sucks here as math.Pow works with floats only
	//why!? I have no idea. Some poking around on the web says building a loop that works with
	//ints is better and really this is not going to be my performance issue so I am just going
	//to do that
	total := 1
	for i := 1; i <= exponent; i++ {
		total *= 2
	}
	exponentialRetryDelayString := fmt.Sprintf("%ds", total)
	return time.ParseDuration(exponentialRetryDelayString)
}
