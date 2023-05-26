package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/akamensky/argparse"

	"galtashma/editor-server/server"
)

func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return dir
}

func main() {
	// Create a new parser object
	parser := argparse.NewParser("enum-example", "Example of parsing an enum flag")

	connectionType := parser.Selector("p", "protocol", []string{"grpc", "connect"}, &argparse.Options{
		Required: true,
		Help:     "Type of connection (grpc or connect)",
		Default:  "grpc",
	})

	rootDirectory := parser.String("r", "root", &argparse.Options{
		Required: false,
		Help:     "Root Git directory",
		Default:  getCurrentDirectory(),
	})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Println(parser.Usage(err))
		return
	}

	switch *connectionType {
	case "grpc":
		if err := server.StartGRPCServer(*rootDirectory, "localhost:50051"); err != nil {
			panic(err)
		}

	case "connect":
		server.StartConnectServer(*rootDirectory, "localhost:8080")
	default:
		panic("invalid connection type")
	}
}
