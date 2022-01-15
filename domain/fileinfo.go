package domain

//FileInfo holds data about a single file that might be transfered
type FileInfo struct {

	//Fullname is the name and path of an object (dir, file) on the local filesystem
	FullName string

	//Size is the size in bytes of the file
	Size int64

	//Excluded is true if a rule has excluded this object from backup
	Excluded bool

	//Hash is set to the MD5 hash of the object. Used to confirm the object was sent to AWS as expected
	Hash string

	//HashSuccess is set true if the local object has been hashed without error
	HashSuccess bool

	//StorageSuccess is set true if the local object has been confirmed to be stored in AWS S3
	StorageSuccess bool
}

//Copy returns a deep copy of the current FileINfo object
func (fi FileInfo) Copy() *FileInfo {
	return &FileInfo{
		FullName:       fi.FullName,
		Size:           fi.Size,
		Excluded:       fi.Excluded,
		Hash:           fi.Hash,
		HashSuccess:    fi.HashSuccess,
		StorageSuccess: fi.StorageSuccess,
	}
}
