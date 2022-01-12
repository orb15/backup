package domain

import (
	"regexp"
)

//Exclusion holds data about a directory or file exclusion rule
type Exclusion struct {

	//Id is the rule id
	Id int

	//Raw is the raw regex pattern string read from the exclusions file
	Raw string

	//Regex is the compiled regex version of the raw pattern
	Regex *regexp.Regexp
}
