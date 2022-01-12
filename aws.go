package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"backup/domain"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const (
	dryrunSampleFileListLength = 25
	awsRegionUSEast1           = "us-east-1"
)

//this map maps the simple region (eg "us-east-2") to a an enumerated type in the Go SDK. It would appear
//setting region in the s3Client is not enough. Extend this map to handle other regions EXCEPT "us-east-1"
//which acts as kind of a catch-all region where no such specification is required
var awsRegionToLocationConstraintMap = map[string]s3types.BucketLocationConstraint{
	"us-east-2": s3types.BucketLocationConstraintUsEast2,
}

func writeObjectsToAws(appConfig domain.Config, objectsList []*domain.FileInfo) error {

	//create a context to use with all AWS calls
	ctx := context.Background()

	//S3 config uses credentials and named profile from $HOME/.aws directory
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(appConfig.AwsProfile()),
		config.WithRegion(appConfig.Region()))
	if err != nil {
		return fmt.Errorf("AWS config failed: %v", err)
	}
	s3Client := s3.NewFromConfig(cfg)

	//do things differently on a dryrun
	if appConfig.Dryrun() {
		return handleDryrun(ctx, s3Client, appConfig, objectsList)
	}

	//prepare to create the bucket in the current region. Deal with AWS not respecting the region in the Client
	//and the fact that us-east-1 is a default that does not use the LocationConstraint mechanism. Fun!
	bucket := appConfig.Bucket()
	region := appConfig.Region()
	cbInput := &s3.CreateBucketInput{
		Bucket: &bucket,
	}

	if region != awsRegionUSEast1 {
		//look up the location constraint based on the region we are using
		locationConstraint, found := awsRegionToLocationConstraintMap[appConfig.Region()]
		if !found {
			return fmt.Errorf("no coorisponding LocationConstraint for region: %s. Extend the map in aws.go", appConfig.Region())
		}
		cbInput.CreateBucketConfiguration = &s3types.CreateBucketConfiguration{LocationConstraint: locationConstraint}
	}

	//actually create the bucket
	_, err = s3Client.CreateBucket(ctx, cbInput)
	if err != nil {
		return fmt.Errorf("unable to create bucket: %s error: %v", bucket, err)
	}

	//actually write the objects to s3
	writeAllObjectsToS3(ctx, s3Client, appConfig, objectsList)

	return nil
}

func writeAllObjectsToS3(ctx context.Context, s3Client *s3.Client, appConfig domain.Config, objectsList []*domain.FileInfo) {
	logger := appConfig.Logger()
	defer logger.Sync()

	logger.Infow("preparing to store objects", "storageRoutineCount", appConfig.HashRoutinesCount(), "meta", domain.Chat)
	storeStart := time.Now()

	//the channel that will carry all data to the routines - size it to handle the data we will put in
	channel := make(chan *domain.FileInfo, len(objectsList))

	//load the channel with objects to process
	for _, fi := range objectsList {

		//do not attempt to store files that have not been / were not successfully hashed
		if !fi.HashSuccess {
			logger.Infow("skipping un-hashed file", "path", fi.FullName, "meta", domain.Aws)
			continue
		}
		channel <- fi
	}

	//close the channel so all consumers know when the work is done
	close(channel)

	//launch multiple go routines to store the objects. use waitgroup to halt main thread until all
	//routines are finished
	var wg sync.WaitGroup
	for i := 0; i < appConfig.StorageRoutinesCount(); i++ {
		wg.Add(1)
		go storeFilesInChannel(ctx, s3Client, appConfig, objectsList, channel, &wg)
	}

	logger.Infow("waiting for storing to complete...", "meta", domain.Chat)
	wg.Wait()

	storeTime := time.Since(storeStart)
	logger.Infow("storing is complete", "totalTime", storeTime, "meta", domain.Stat)
}

func storeFilesInChannel(ctx context.Context, s3Client *s3.Client, appConfig domain.Config, objectsList []*domain.FileInfo, ch chan *domain.FileInfo, wg *sync.WaitGroup) {
	logger := appConfig.Logger()
	defer logger.Sync()
	defer wg.Done()

	//track file and storaged-related errors and shut down this routine if excessive errors occur
	//this is just a quick failure in case there is a systemic problem somewhere - it allows
	//the routine to give up under the assumption that a systemic issue will cause all other
	//routines issues as well and there is no sense in continuing to try to open ~25-50K files
	//under such circumstances
	errCount := 0
	maxAllowedErrors := appConfig.MaxStorageChannelErrorCount()

	filesProcessed := 0
	for fi := range ch {

		filesProcessed++
		filename := fi.FullName

		//open the file
		f, err := os.Open(filename)
		if err != nil {
			errCount++
			logger.Errorw("failed to open file for storage", "path", fi.FullName, "err", err, "meta", domain.Err)
			fi.StorageSuccess = false
		} else {

			//prep the call to AWS s3
			bucket := appConfig.Bucket()
			key := toKey(filename)
			poi := &s3.PutObjectInput{
				Bucket:     &bucket,
				Key:        &key,
				Body:       f,
				ContentMD5: &fi.Hash,
			}

			//retry AWS a few times using a 2^n exponential backoff where n is
			//the number of failures that have happened for this file
			storageErrorCount := 0
			var storageErr error
			allowedStorageAttempts := appConfig.StorageRetryCount()
			if allowedStorageAttempts <= 0 {
				allowedStorageAttempts = 1
			}

			//while we have not hit our max error threshold, attempt a send to S3
			for storageErrorCount < allowedStorageAttempts {

				//send to S3
				_, storageErr = s3Client.PutObject(ctx, poi)

				//storage error
				if storageErr != nil {

					//increase failure count
					storageErrorCount++
					logger.Debugw("putObject attempt failed", "path", filename, "failCount", storageErrorCount, "meta", domain.Aws)

					//give up & leave retry loop - no sense in mucking about with retries, we have failed
					if storageErrorCount >= allowedStorageAttempts {
						break
					}

					//a retry is possible - calculate a Duration based on exponential backoff (thanks Go math lib for sucking here!)
					exponentialRetryDelayString := fmt.Sprintf("%1.fs", math.Pow(2, float64(storageErrorCount)))
					d, err := time.ParseDuration(exponentialRetryDelayString)
					if err != nil { //failed to parse the duration - should not happen, right? Right?
						logger.Errorw("failed to parse exponential duration", "duration", exponentialRetryDelayString, "err", err, "meta", domain.Err)
					} else {
						time.Sleep(d) //sleep this thread and retry
					}
				} else { //storage success, leave the retry loop
					break
				}
			}

			//we still failed after retries, mark this as a true failure
			if storageErr != nil {
				errCount++
				logger.Errorw("failed to store file after exhausting retries", "path", filename, "err", storageErr, "meta", domain.Err)
				fi.StorageSuccess = false
			} else {
				fi.StorageSuccess = true
			}

			err := f.Close()
			if err != nil {
				logger.Warnw("failed to close file after storing", "path", fi.FullName, "meta", domain.Aws)
			}
		}

		//exit on excessive errors
		if errCount > maxAllowedErrors {
			logger.Errorw("storage routine exceeded max error count. Shutting it down", "maxAllowedErrors", maxAllowedErrors, "meta", domain.Aws)
			break
		}

		//note each 100 files this routine handles
		if filesProcessed == 100 {
			logger.Debugw("a storage routine has processed 100 files", "meta", domain.Chat)
			filesProcessed = 0
		}

	}

}

//critical function here - change a win file name (eg E:\\foo\\bar) into something S3 will use to build folders in the console (E:->foo->bar)
func toKey(filename string) string {
	return strings.ReplaceAll(filename, "\\", "/")
}

func handleDryrun(ctx context.Context, s3Client *s3.Client, appConfig domain.Config, objectsList []*domain.FileInfo) error {
	logger := appConfig.Logger()
	defer logger.Sync()

	logger.Infow("beginning AWS dryrun...", "meta", domain.Chat)

	var sb strings.Builder

	//config dump
	sb.WriteString("\n")
	sb.WriteString("Current Configuration\n")
	sb.WriteString("---------------------\n")
	sb.WriteString(appConfig.String())

	//Try to contact AWS and get a bucket list
	lbOutput, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return fmt.Errorf("dryrun Error: unable to list AWS buckets: %v", err)
	}

	found := false
	sb.WriteString("\n")
	sb.WriteString("AWS Bucket Listing\n")
	sb.WriteString("---------------------\n")
	for _, b := range lbOutput.Buckets {
		sb.WriteString(fmt.Sprintf("  %s\n", *b.Name))
		if *b.Name == appConfig.DryrunBucket() {
			found = true
		}
	}

	//note the existance of the dryrun bucket
	sb.WriteString("Successfully located dryrun bucket: ")
	if found {
		sb.WriteString("true\n")
	} else {
		sb.WriteString("false\n")
		return fmt.Errorf("dryrun Error: unable to locate dryrun bucket: %s", appConfig.DryrunBucket())
	}

	//print out a selection of the files to be transferred
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("Sample File List [%d of %d total files]\n", dryrunSampleFileListLength, len(objectsList)))
	sb.WriteString("---------------------------------------\n")
	for i := 0; i < dryrunSampleFileListLength; i++ {
		sb.WriteString(fmt.Sprintf("  %s\n", objectsList[i].FullName))
	}

	fmt.Println(sb.String())

	logger.Infow("AWS dryrun complete", "meta", domain.Chat)
	return nil
}
