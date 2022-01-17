package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"backup/domain"
)

//walks each path in each directory to be archived and builds a list of files that need to be backed up
func buildFileList(appConfig domain.Config) ([]*domain.FileInfo, error) {
	logger := appConfig.Logger()
	defer logger.Sync()

	allInfo := make([]*domain.FileInfo, 0, appConfig.FileCountEstimate())

	//for each top-level path
	for _, pth := range appConfig.BasePaths() {

		logger.Infow("beginning examination of top level path", "path", pth, "meta", domain.Chat)

		//walk that path and call the anaon function to process that path and it's children
		err := filepath.Walk(pth,
			func(path string, info os.FileInfo, err error) error {

				//can happen under special circumstances where Walk calls this func with err set- see API docs for these rare cases
				if err != nil {
					return err
				}

				//build struct about this file/dir - assume it is excluded
				newFileData := &domain.FileInfo{
					FullName: path,
					Size:     info.Size(),
					Excluded: true,
				}

				//determine if we should skip the file or directory. Note that we _always_ skip directories but we
				//need to first determine if we are skipping the directory because it has bene excluded
				//(and thus all contents must also be excluded) or if we are skipping the directory simply because it
				//is a directory. The subtlety here is in what we return. A return of filepath.SkipDir indicates
				//the file walker should not descend into the directory's children while a return of nil indicates that
				//the walker should continue processing children
				if info.IsDir() && skipThisObject(appConfig, path, info) {
					allInfo = append(allInfo, newFileData)
					return filepath.SkipDir //do not process this directory (or its children) further
				} else if skipThisObject(appConfig, path, info) {
					allInfo = append(allInfo, newFileData)
					return nil //not interested in this file
				}

				//this is a dir we are interested in, but since it isn't a file, just continue
				//(and by continue I mean descend into this dir)
				if info.IsDir() {
					allInfo = append(allInfo, newFileData)
					return nil
				}

				//we want this file, add it to the list
				newFileData.Excluded = false
				allInfo = append(allInfo, newFileData)
				return nil
			})

		if err != nil {
			return nil, err
		}
	}

	return allInfo, nil
}

//determines if any object (file) should be excluded from the backup because of a rule
func skipThisObject(appConfig domain.Config, path string, info os.FileInfo) bool {
	logger := appConfig.Logger()
	defer logger.Sync()

	//RULE: skip directories that start with .
	if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
		logger.Debugw("hardcoded exclusion - directory begins with dot", "path", path, "meta", domain.Exclude)
		return true
	}

	//test each regex rule in the exclusions list and if one applies, return false
	for _, exclusion := range appConfig.Exclusions() {
		if exclusion.Regex.MatchString(path) {
			logger.Debugw("rule exclusion", "path", path, "isDir", info.IsDir(), "rule id", exclusion.Id, "meta", domain.Exclude)
			return true
		}
	}

	return false
}

//reprocesses a JSON file listing files that failed to transfer on the last run
func buildReprocessingList(appConfig domain.Config) ([]*domain.FileInfo, error) {

	//read JSON file
	jsonBytes, err := os.ReadFile(appConfig.FailuresFilepath())
	if err != nil {
		return nil, fmt.Errorf("unable to read JSON failures file: %s because: %v", appConfig.FailuresFilepath(), err)
	}

	//unmarshall
	var failures domain.BackupFailures
	err = json.Unmarshal(jsonBytes, &failures)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON failures file: %s because: %v", appConfig.FailuresFilepath(), err)
	}

	//no work to do
	if !failures.HasFailures {
		return nil, nil
	}

	//get confirmation unless bypassed by CLI opts
	if !appConfig.NoConfirm() {
		runMenu := true
		var choice int
		for runMenu {
			fmt.Println()
			fmt.Printf("File Reprocessing Menu for Backup Performed On: %s\n", failures.DateCreated)
			fmt.Println()
			fmt.Println("1: List Files to reprocess")
			fmt.Println("2: Reprocess now")
			fmt.Println("3: Cancel Reprocessing")
			fmt.Println()
			fmt.Println("Enter ")
			_, err := fmt.Scanf("%d", &choice)
			if err != nil {
				continue
			}
			switch choice {
			case 1:
				fmt.Println()
				for _, f := range failures.FailedPaths {
					fmt.Println(f)
				}
				fmt.Println()
			case 2:
				runMenu = false
				continue
			case 3:
				return nil, nil
			default:
				continue
			}
		}
	}

	//build file list. Wipe out the old hash and file size
	fileData := make([]*domain.FileInfo, 0, len(failures.FailedPaths))

	for _, f := range failures.FailedPaths {
		fi := &domain.FileInfo{
			FullName:       f.FullName,
			Hash:           "",
			Excluded:       false,
			HashSuccess:    false,
			StorageSuccess: false,
		}

		fileInfo, err := os.Stat(f.FullName)
		if err != nil {
			return nil, fmt.Errorf("unable to stat file: %s because: %v", f.FullName, err)
		}
		fi.Size = fileInfo.Size()
		fileData = append(fileData, fi)
	}

	return fileData, nil
}
