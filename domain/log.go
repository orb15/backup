package domain

//these constants provide a log message 'meta-type' (think error code) to allow
//accurate and graceful post-processing of the logs or filtering with a tool like jq
const (

	//Chat is used when the message is purely informational
	Chat = "CHAT"

	//Core is used when the message is a core message - usually a fatal or error level message that doesn't fall into any other catagory
	Core = "CORE"

	//Err is used when the message contains an error reported from elsewhere in the system. They always contain an "err" element
	Err = "ERR"

	//Hash is used for important hash-related messages that dont fall into another catagory
	Hash = "HASH"

	//Stat is slightly more important than Chat in that some useful metric is conveyed
	Stat = "STAT"

	//Exclude is used for exclusions - rules that prevent a folder or file from being proccessed
	Exclude = "EXCLUSION"

	//Aws is used for AWS-related messages that don't fall into another catagory
	Aws = "AWS"
)
