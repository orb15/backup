package main

import (
	"time"

	"backup/domain"
)

//calulates stats regarding which files will be stored/excluded. Also returns a list of files to be stored
func displayFileStats(appConfig domain.Config, objectsList []*domain.FileInfo) []*domain.FileInfo {

	logger := appConfig.Logger()
	defer logger.Sync()

	logger.Infow("total number of objects to potentially store", "count", len(objectsList), "meta", domain.Stat)

	//create a list of "storable objects" since we are iterating the list anyway
	saveThese := make([]*domain.FileInfo, 0)

	//more metrics
	save := 0
	var saveSize int64
	reject := 0
	var rejectSize int64
	for _, o := range objectsList {
		if o.Excluded {
			reject++
			rejectSize += o.Size
		} else {
			save++
			saveSize += o.Size
			saveThese = append(saveThese, o)
		}
	}
	logger.Infow("objects to be saved metrics", "count", save, "totalSize", saveSize, "meta", domain.Stat)
	logger.Infow("objects to be excluded metrics", "count", reject, "totalSize", rejectSize, "meta", domain.Stat)

	return saveThese
}

//counts and displays files that failed hashing
func displayBadHashes(appConfig domain.Config, objectsList []*domain.FileInfo) bool {
	defer appConfig.Logger().Sync()

	failedHashCount := 0
	for _, fi := range objectsList {
		if !fi.Excluded && !fi.HashSuccess {
			failedHashCount++
		}
	}
	appConfig.Logger().Infow("failed hash count", "count", failedHashCount, "meta", domain.Stat)

	//determine if too many hashing failures have occured to safely continue
	return failedHashCount >= appConfig.MaxAllowedHashFailures()
}

//displays stats regarding files that were stored or failed to store. Also returns details about all failed files
func displayStorageStats(appConfig domain.Config, objectsList []*domain.FileInfo) *domain.BackupFailures {

	logger := appConfig.Logger()
	defer logger.Sync()

	//prep JSON struct to hold failure data
	failures := &domain.BackupFailures{
		DateCreated: time.Now().Format(time.RFC822),
		Bucket:      appConfig.Bucket(),
		HasFailures: false,
		FailedPaths: make([]*domain.FileInfo, 0),
	}

	success := 0
	failed := 0
	for _, o := range objectsList {
		if o.Excluded {
			continue
		}
		if o.StorageSuccess {
			success++
		} else {
			failed++
			failures.HasFailures = true
			failed := o.Copy()
			failures.FailedPaths = append(failures.FailedPaths, failed)
		}
	}
	logger.Infow("number of objects successfully stored", "count", success, "meta", domain.Stat)
	logger.Infow("number of storage failures", "count", failed, "meta", domain.Stat)

	return failures
}
