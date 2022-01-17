package domain

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	defaultDryrunBucket         = "dryrun-2155"
	defaultExclusionsFile       = "exclusions.txt"
	defaultBackupDirectivesFile = "backup.txt"
	defaultFailureOutputFile    = "failures.json"
	defaultSharedProfile        = "s3-only"
	defaultAwsRegion            = "us-east-2"

	defaultFileCountEstimate = 25000

	defaultHashRoutines                   = 100
	defaultHashEffortChannelMaxErrorCount = 25
	defaultAllowedFailedHashCount         = 25

	defaultStorageRoutines            = 100
	defaultStorageChannelMaxErrorRate = 25
	defaultStorageRetryCount          = 5
)

//Config holds core info about the app
type Config interface {
	DryrunBucket() string
	Region() string
	AwsProfile() string
	Bucket() string

	FailuresFilepath() string

	Dryrun() bool
	Reprocess() bool
	NoConfirm() bool
	Logger() *zap.SugaredLogger

	Exclusions() []*Exclusion
	BasePaths() []string
	FileCountEstimate() int

	HashRoutinesCount() int
	MaxHashChannelErrorCount() int
	MaxAllowedHashFailures() int

	StorageRoutinesCount() int
	MaxStorageChannelErrorCount() int
	StorageRetryCount() int

	String() string
}

type appConfig struct {
	dryrunBucket                  string
	region                        string
	awsProfile                    string
	bucket                        string
	dryrun                        bool
	reprocess                     bool
	noConfirm                     bool
	logger                        *zap.SugaredLogger
	exclusionsFile                string
	backupFile                    string
	failuresFile                  string
	exclusions                    []*Exclusion
	basePaths                     []string
	fileCountEstimate             int
	hashRoutines                  int
	maxHashChannelErrorAllowed    int
	allowedHashFailCount          int
	storageRoutines               int
	maxStorageChannelErrorAllowed int
	storageRetryCount             int
}

//NewConfig does just what it says on the tin
func NewConfig(cmdOpts *CommandOpts) (*appConfig, error) {
	return newConfig(cmdOpts)
}

//DryrunBucket returns the name of the bucket used during dryruns
func (ac *appConfig) DryrunBucket() string {
	return ac.dryrunBucket
}

//Region returns the AWS region where the buckets will be stored
func (ac *appConfig) Region() string {
	return ac.region
}

//AwsProfile returns the AWS profile to use for accessing S3 (see $HOME/.aws)
func (ac *appConfig) AwsProfile() string {
	return ac.awsProfile
}

//Bucket returns the name of the bucket to which all objects will be stored
func (ac *appConfig) Bucket() string {
	return ac.bucket
}

//Dryrun returns true if the user is asking for a dry run
func (ac *appConfig) Dryrun() bool {
	return ac.dryrun
}

//Reprocess returns true if the user is asking to reprocess previously-failed files
func (ac *appConfig) Reprocess() bool {
	return ac.reprocess
}

//NoConfirm returns true if, during reprocessing, the confirmationmenu should be skipped
func (ac *appConfig) NoConfirm() bool {
	return ac.noConfirm
}

//Logger returns the logger
func (ac *appConfig) Logger() *zap.SugaredLogger {
	return ac.logger
}

//FailuresFilename returns the path  of the file where failures will be stored
func (ac *appConfig) FailuresFilepath() string {
	return ac.failuresFile
}

//Exclusions returns all exclusions in the exclusions file
func (ac *appConfig) Exclusions() []*Exclusion {
	return ac.exclusions
}

//BasePaths returns the base drive and directory where backups begin
func (ac *appConfig) BasePaths() []string {
	return ac.basePaths
}

//FileCountEstimate returns the estimated number of files that will be sent to S3
func (ac *appConfig) FileCountEstimate() int {
	return ac.fileCountEstimate
}

//HashRoutinesCount returns the number of go routines to use when hashing all files to be transferred
func (ac *appConfig) HashRoutinesCount() int {
	return ac.hashRoutines
}

//MaxHashChannelErrorCount returns the max number of errors a hash routine can experience before stopping that routine
func (ac *appConfig) MaxHashChannelErrorCount() int {
	return ac.maxHashChannelErrorAllowed
}

//MaxAllowedHashFailures returns the max number of errors allowed across all routines before overall S3 operations are aborted
func (ac *appConfig) MaxAllowedHashFailures() int {
	return ac.allowedHashFailCount
}

//StorageRoutinesCount returns the number of go routines to use when storing files to aws
func (ac *appConfig) StorageRoutinesCount() int {
	return ac.storageRoutines
}

//MaxStorageChannelErrorCount returns the max number of errors a storage routine can experience before stopping that routine
func (ac *appConfig) MaxStorageChannelErrorCount() int {
	return ac.maxStorageChannelErrorAllowed
}

//StorageRetryCount returns the number of retries (if any) PutObject should be called before giving up
func (ac *appConfig) StorageRetryCount() int {
	return ac.storageRetryCount
}

//Reads exclusions from a flat file. Each line is a regex indicating a location in the basedir
//to be excluded
func (ac *appConfig) readExclusions() ([]*Exclusion, error) {

	logger := ac.logger
	defer logger.Sync()

	file, err := os.Open(ac.exclusionsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open exclusions file: %s", ac.exclusionsFile)
	}
	defer func() {
		if err = file.Close(); err != nil {
			logger.Errorw("failed to close exclusions file on deferred close\n", "path", ac.exclusionsFile, "meta", Core)
		}
	}()

	scanner := bufio.NewScanner(file)
	exclusions := make([]*Exclusion, 0)

	index := 0
	for scanner.Scan() {

		line := scanner.Text()

		//skip some lines in this file
		if strings.HasPrefix(line, "#") {
			continue
		}
		if len(line) == 0 {
			continue
		}

		//assign this rule an index
		index++

		//convert to regex & add it to the list of rules
		rgx, err := regexp.Compile(line)
		if err != nil {
			return nil, fmt.Errorf("failed to compile exclusion: '%s' with error: %v", line, err)
		}
		ex := &Exclusion{
			Id:    index,
			Raw:   line,
			Regex: rgx,
		}
		logger.Debugw("adding exclusion rule", "id", index, "rule", line, "meta", Exclude)
		exclusions = append(exclusions, ex)
	}

	logger.Infow("added exclusion rules from exclusions file", "ruleCount", len(exclusions), "path", ac.exclusionsFile, "meta", Exclude)
	return exclusions, nil
}

//reads which directories to backup from a file
func (ac *appConfig) readBackupDirectives() error {
	logger := ac.logger
	defer logger.Sync()

	file, err := os.Open(ac.backupFile)
	if err != nil {
		return fmt.Errorf("unable to open backup directives file: %s", ac.backupFile)
	}
	defer func() {
		if err = file.Close(); err != nil {
			logger.Errorw("failed to close backup file on deferred close\n", "path", ac.backupFile, "meta", Core)
		}
	}()

	//create a regex for validating the file contents - checks for drive letter and proper slash (eg C:\)
	const drivePattern = `^[A-Z]:\\.*`
	regx := regexp.MustCompile(drivePattern)

	paths := make([]string, 0)

	index := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		line := scanner.Text()
		line = strings.TrimSpace(line)

		//skip some lines in this file
		if strings.HasPrefix(line, "#") {
			continue
		}
		if len(line) == 0 {
			continue
		}
		index++

		//ensure drive letter, colon and double slashes
		if !regx.MatchString(line) {
			fmt.Println(line)
			return fmt.Errorf(`%s line: %d defines invalid backup location. Must be of the format: C:\... `, ac.backupFile, index)
		}

		//check to make sure slashes are always in pairs
		// slashCount := strings.Count(line, `\`)
		// if (slashCount % 2) != 0 {
		// 	return fmt.Errorf(`%s line: %d defines invalid backup location. All slashes must be double slashes (\\ not \)`, ac.backupFile, index)
		// }

		paths = append(paths, line)
	}

	if len(paths) == 0 {
		return fmt.Errorf("no backup directories specified in file: %s. Nothing to do", ac.backupFile)
	}

	//assign the paths to the config object
	ac.basePaths = paths

	logger.Infow("created backup directory list from file", "path", ac.backupFile, "directoryCount", len(paths), "meta", Chat)
	return nil
}

//stringify the config for display
func (ac *appConfig) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Dryrun Bucket: %s\n", ac.dryrunBucket))
	sb.WriteString(fmt.Sprintf("Dryrun Enabled: %t\n", ac.dryrun))
	sb.WriteString(fmt.Sprintf("Exclusions File: %s\n", ac.exclusionsFile))
	sb.WriteString(fmt.Sprintf("Failures File: %s\n", ac.failuresFile))
	sb.WriteString(fmt.Sprintf("Exclusions Count: %d\n", len(ac.exclusions)))
	sb.WriteString(fmt.Sprintf("Base Paths: %s\n", ac.basePaths))
	sb.WriteString(fmt.Sprintf("AWS Profile: %s\n", ac.awsProfile))
	sb.WriteString(fmt.Sprintf("AWS Region: %s\n", ac.region))
	sb.WriteString(fmt.Sprintf("Target Bucket: %s\n", ac.bucket))
	sb.WriteString(fmt.Sprintf("Number of Hash Routines: %d\n", ac.hashRoutines))
	sb.WriteString(fmt.Sprintf("Number of Storage Routines: %d\n", ac.storageRoutines))
	sb.WriteString(fmt.Sprintf("Storage Retry Count: %d\n", ac.storageRetryCount))

	return sb.String()
}

//create a config based on defaults then override those defaults with sommand line opts
func newConfig(cmdOpts *CommandOpts) (*appConfig, error) {

	//create default config
	c := &appConfig{
		dryrunBucket:                  defaultDryrunBucket,
		region:                        defaultAwsRegion,
		awsProfile:                    defaultSharedProfile,
		bucket:                        makeUniqueBucketName(),
		dryrun:                        cmdOpts.Dryrun,
		reprocess:                     cmdOpts.Reprocess,
		noConfirm:                     cmdOpts.NoConfirm,
		exclusionsFile:                defaultExclusionsFile,
		backupFile:                    defaultBackupDirectivesFile,
		failuresFile:                  defaultFailureOutputFile,
		fileCountEstimate:             defaultFileCountEstimate,
		hashRoutines:                  defaultHashRoutines,
		maxHashChannelErrorAllowed:    defaultHashEffortChannelMaxErrorCount,
		allowedHashFailCount:          defaultAllowedFailedHashCount,
		storageRoutines:               defaultStorageRoutines,
		maxStorageChannelErrorAllowed: defaultStorageChannelMaxErrorRate,
		storageRetryCount:             defaultStorageRetryCount,
	}

	//create logger with INFO level enabled
	zapConfig := zap.Config{
		Encoding:    "json",
		OutputPaths: []string{"stderr"},
		Level:       zap.NewAtomicLevelAt(zapcore.InfoLevel),
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:  "message",
			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,
			TimeKey:     "time",
			EncodeTime:  zapcore.ISO8601TimeEncoder,
		},
	}

	//if debug logging is requested, set the logger to that level instead
	if cmdOpts.UseDebugLogger {
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}

	//build a sugared logger from the config
	zapLogger, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}
	c.logger = zapLogger.Sugar()
	defer c.logger.Sync()
	c.logger.Infow("zap logger configured and available", "meta", Chat)

	//read and compile regex exclusions from flat file
	exclusions, err := c.readExclusions()
	if err != nil {
		return nil, err
	}
	c.exclusions = exclusions

	//read backup directives from file
	err = c.readBackupDirectives()
	if err != nil {
		return nil, err
	}

	return c, nil
}

//create a new bucket name based on date and UUID
func makeUniqueBucketName() string {
	dateName := time.Now().Format("02Jan2006")
	dateName = strings.ToLower(dateName)
	uid := uuid.New()
	return dateName + "-" + uid.String()
}
