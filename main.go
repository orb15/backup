package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"backup/domain"
)

func main() {

	//flags handling
	debugLoggingPtr := flag.Bool("debug", false, "set to enable debug logging")
	dryrunPtr := flag.Bool("dryrun", false, "set to enable dryrun (no aws calls)")
	flag.Parse()

	cmdOpts := &domain.CommandOpts{
		UseDebugLogger: *debugLoggingPtr,
		Dryrun:         *dryrunPtr,
	}

	//create config with defaults overriden by app params
	appConfig := buildConfig(cmdOpts)

	//prep the logger now that we know it is ready
	logger := appConfig.Logger()
	defer logger.Sync()

	//build a list of all possible files
	allObjectsList, err := buildFileList(appConfig)
	if err != nil {
		logger.Fatalw("error when building file list", "err", err, "meta", domain.Err)
	}

	//provide some basic stats on the amount of files and data to transfer/exclude and fetch a list
	//of files we actually will process (eg trim files we are excluding from our list)
	objectsToStore := displayFileStats(appConfig, allObjectsList)

	//hash all files we are planning to transfer - skip this if we are in dry run
	if !appConfig.Dryrun() {

		hashAllFiles(appConfig, objectsToStore)

		//display a count of files that failed to hash for some reason and determine if we should continue
		tooManyFailedHashes := displayBadHashes(appConfig, objectsToStore)
		if tooManyFailedHashes {
			logger.Fatalw("hash calculation failures exceed allowable maximum. Aborting", "configuredMax", appConfig.MaxAllowedHashFailures(), "meta", domain.Core)
		}
	} else {
		logger.Infow("skipping file hashing because of dryrun", "meta", domain.Chat)
	}

	//actually write objects to AWS (dry run handled internal to this routine to allow as much execution as possible)
	err = writeObjectsToAws(appConfig, objectsToStore)
	if err != nil {
		logger.Fatalw("critical AWS failure", "err", err, "meta", domain.Err)
	}

	//display a count of files that failed to be stored
	var failedFilesDetails string
	if !appConfig.Dryrun() {
		failedFilesDetails = displayStorageStats(appConfig, allObjectsList)
	}

	//display total run time
	logger.Infow("total execution time", "time", time.Since(appConfig.Start()), "meta", domain.Stat)

	//after all logging, print a list of failed files
	if !appConfig.Dryrun() && len(failedFilesDetails) > 0 {
		fmt.Println(failedFilesDetails)
	}
}

func buildConfig(cmdOpts *domain.CommandOpts) domain.Config {

	//build a new app config
	cfg, err := domain.NewConfig(cmdOpts)
	if err != nil {
		fmt.Printf("FATAL: configuration error: %v\n", err)
		os.Exit(1)
	}

	return cfg
}
