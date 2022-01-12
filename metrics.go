package main

import (
	"fmt"
	"strings"

	"backup/domain"
)

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

func displayStorageStats(appConfig domain.Config, objectsList []*domain.FileInfo) string {

	logger := appConfig.Logger()
	defer logger.Sync()

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("Failed Files Listing\n")
	sb.WriteString("--------------------\n")

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
			sb.WriteString(fmt.Sprintf("%s\n", o.FullName))
		}
	}
	logger.Infow("number of objects successfully stored", "count", success, "meta", domain.Stat)
	logger.Infow("number of storage failures", "count", failed, "meta", domain.Stat)

	sb.WriteString("\n")
	return sb.String()
}
