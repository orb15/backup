package domain

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
)

const (
	defaultBucket         = "homebackup-2155"
	defaultIsLogging      = true
	defaultExclusionsFile = "exclusions.txt"
	defaultBaseDrive      = "E"
	defaultBaseDir        = "Misc"
	defaultSharedProfile  = "s3-only"
)

//Config is the config
type Config interface {
	Bucket() string
	IsLogging() bool
	Logger() log.Logger
	BaseDir() string
	AwsProfile() string
	Prefix() string
	BasePath() string
}

type appConfig struct {
	bucket       string
	isLogging    bool
	exclusions   []*regexp.Regexp
	logger       log.Logger
	baseDir      string
	basePath     string
	awsProfile   string
	objectPrefix string
}

//NewConfig does just what it says on the tin
func NewConfig(prefix string) (*appConfig, error) {
	return defaultConfig(prefix)
}

//Bucket returns the target bucket
func (ac *appConfig) Bucket() string {
	return ac.bucket
}

//Logger returns the logger
func (ac *appConfig) Logger() log.Logger {
	return ac.logger
}

//IsLogging returns true if logging enabled
func (ac *appConfig) IsLogging() bool {
	return ac.isLogging
}

//BaseDir returns the base directory where backups begin
func (ac *appConfig) BaseDir() string {
	return ac.baseDir
}

//BasePath returns the base drive and directory where backups begin
func (ac *appConfig) BasePath() string {
	return ac.basePath
}

//AwsProfile returns the AWS profile to use for accessing S3 (see $HOME/.aws)
func (ac *appConfig) AwsProfile() string {
	return ac.awsProfile
}

//Prefix returns the prefix that will be prepended to every object stored in S3 on a given run
func (ac *appConfig) Prefix() string {
	return ac.objectPrefix
}

//Reads exclusions from a flat file. Each line is a regex indicating a location in the basedir
//to be excluded
func (ac *appConfig) readExclusions() ([]*regexp.Regexp, error) {
	file, err := os.Open(defaultExclusionsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open exclusions file: %s", defaultExclusionsFile)
	}
	defer func() {
		if err = file.Close(); err != nil {
			fmt.Printf("failed to close exclusions file: %s on defered close\n", defaultExclusionsFile)
		}
	}()

	scanner := bufio.NewScanner(file)
	regexes := make([]*regexp.Regexp, 0)

	for scanner.Scan() {
		line := scanner.Text()
		regex, err := regexp.Compile(line)
		if err != nil {
			return nil, fmt.Errorf("failed to compile exclusion: '%s' with error: %v", line, err)
		}
		regexes = append(regexes, regex)
	}

	return regexes, nil
}

func defaultConfig(prefix string) (*appConfig, error) {

	c := &appConfig{
		bucket:       defaultBucket,
		isLogging:    defaultIsLogging,
		baseDir:      defaultBaseDir,
		awsProfile:   defaultSharedProfile,
		objectPrefix: prefix,
	}

	//read exclusions from flat file
	exclusions, err := c.readExclusions()
	if err != nil {
		return nil, err
	}

	//create base path
	c.basePath = defaultBaseDrive + ":\\" + c.baseDir

	c.exclusions = exclusions
	return c, nil
}
