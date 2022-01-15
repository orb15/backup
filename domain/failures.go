package domain

//BackupFailures holds information about failed file transfers for a given transfer/backup attempt
type BackupFailures struct {

	//Bucket is the name of the bucket to which the files should have been transferred
	Bucket string `json:"bucket"`

	//HasFailures is true when there is at least one failure
	HasFailures bool `json:"hasFailures"`

	//FailedPaths contains the information about each failed file
	FailedPaths []*FileInfo
}
