package main

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	highestReasonableExponentThatWontOverflowInt32 = 46340
)

//create a base64-encoded string of the md5 hash of a file
func hashFile(filename string) (string, error) {

	//open file
	f, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open file for hashing: %s with error %v", filename, err)
	}

	//ensure closure
	defer func() {
		err := f.Close()
		if err != nil {
			//may want to silently fail here as we are opening the file for reading and not closing isnt catastrophic.
			//printing this also may hinder post-processing of JSON logs as this isnt a json message
			//then again, if I am opening a lot of files (and I am) and many aren't closing, I may exhaust file handles
			//and this is something I want to know about so...
			fmt.Printf("failed to close file: %s after hashing. Error: %v\n", filename, err)
		}
	}()

	//hash file to base64 encoded MD5 string
	h := md5.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return "", fmt.Errorf("failed to copy file for hashing: %s with error %v", filename, err)
	}

	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

//convert a duration to a reasonably-looking string
func prettyTime(delta time.Duration) string {
	delta = delta.Round(time.Second)
	return delta.String()
}

//return a time.Duration that represents 2^(exponent) seconds
func calcBackoff(exponent int) (time.Duration, error) {

	//safety & sanity
	if exponent < 0 || exponent > highestReasonableExponentThatWontOverflowInt32 {
		return 0, fmt.Errorf("unsupported exponent value: %d", exponent)
	}
	if exponent == 0 {
		return 1, nil
	}

	//derive an int that is 2^(exponent). Golang sucks here as math.Pow works with floats only
	//why!? I have no idea (actually I do but that is another rant). Some poking around on the
	//web says building a loop that works with ints is better and really this is not going to
	//be my "big performance issue" in this app so I am just going to do that
	total := 1
	for i := 1; i <= exponent; i++ {
		total *= 2
	}
	exponentialRetryDelayString := fmt.Sprintf("%ds", total) //eg 16s for 2^4
	return time.ParseDuration(exponentialRetryDelayString)
}
