package main

import (
	"crypto/md5"
	"encoding/base64"
	"io"
	"os"
	"sync"
	"time"

	"backup/domain"
)

func hashAllFiles(appConfig domain.Config, objectsList []*domain.FileInfo) {

	logger := appConfig.Logger()
	defer logger.Sync()

	logger.Infow("preparing to hash objects", "routineCount", appConfig.HashRoutinesCount(), "meta", domain.Chat)
	hashStart := time.Now()

	//the channel that will carry all data to the routines - size it to handle the data we will put in
	channel := make(chan *domain.FileInfo, len(objectsList))

	//load the channel with objects to process
	for _, fi := range objectsList {
		channel <- fi
	}

	//close the channel so all consumers know when the work is done
	close(channel)

	//launch multiple go routines to calculate hashes. use waitgroup to halt main thread until all
	//routines are finished
	var wg sync.WaitGroup
	for i := 0; i < appConfig.HashRoutinesCount(); i++ {
		wg.Add(1)
		go hashFilesInChannel(appConfig, channel, &wg)
	}

	logger.Infow("waiting for hashing to complete...", "meta", domain.Chat)
	wg.Wait()

	hashTime := time.Since(hashStart)
	logger.Infow("hashing is complete", "hashTotalTime", hashTime, "meta", domain.Chat)
}

func hashFilesInChannel(appConfig domain.Config, ch chan *domain.FileInfo, wg *sync.WaitGroup) {
	logger := appConfig.Logger()
	defer logger.Sync()
	defer wg.Done()

	//track file and hash-related errors and shut down this routine if excessive errors occur
	//this is just a quick failure in case there is a systemic problem somewhere - it allows
	//the routine to give up under the assumption that a systemic issue will cause all other
	//routines issues as well and there is no sense in continuing to try to open ~25-50K files
	//under such circumstances
	errCount := 0
	maxAllowedErrors := appConfig.MaxHashChannelErrorCount()

	filesProcessed := 0
	for fi := range ch {

		filesProcessed++
		filename := fi.FullName

		//open the file and hash it
		f, err := os.Open(filename)
		if err != nil {
			errCount++
			logger.Errorw("failed to open file for hashing", "path", filename, "err", err, "meta", domain.Err)
			fi.HashSuccess = false
		} else {

			h := md5.New()
			if _, err := io.Copy(h, f); err != nil {
				errCount++
				logger.Errorw("failed to copy file for hashing", "path", filename, "err", err, "meta", domain.Err)
				fi.HashSuccess = false
			} else {
				fi.Hash = base64.StdEncoding.EncodeToString(h.Sum(nil))
				fi.HashSuccess = true
			}
			err := f.Close()
			if err != nil {
				logger.Warnw("failed to close file after hashing", "path", fi.FullName, "meta", domain.Hash)
			}
		}

		//exit on excessive errors
		if errCount > maxAllowedErrors {
			logger.Errorw("hash routine exceeded max error count. Shutting it down", "maxAllowedErrors", maxAllowedErrors, "meta", domain.Hash)
			break
		}

		//note each 100 files this routine handles
		if filesProcessed == 100 {
			logger.Debugw("a hashing routine has completed 100 file hashes", "meta", domain.Chat)
			filesProcessed = 0
		}

	}

}
