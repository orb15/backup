package main

import (
	"fmt"
	"os"

	"backup/domain"
)

func main() {
	rawArgs := os.Args[1:]
	buildConfig(rawArgs)

	//S3 config
	//cfg, err := config.LoadDefaultConfig(context.TODO(),
	//config.WithSharedConfigProfile("s3-only"))
}

func buildConfig(args []string) domain.Config {

	if len(args) != 1 {
		fmt.Println("program needs 1 argument to define prefix of all files to be written to S3 this execution")
		os.Exit(1)
	}

	prefix := args[0]

	cfg, err := domain.NewConfig(prefix)
	if err != nil {
		fmt.Printf("configuration error: %v\n", err)
		os.Exit(1)
	}

	return cfg
}
