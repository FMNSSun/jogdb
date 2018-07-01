package main

import . "github.com/FMNSSun/jogdb"
import "github.com/FMNSSun/rndstring"
import "net/http"
import "github.com/gorilla/handlers"
import "os"
import "log"
import "flag"
import "bufio"
import "fmt"
import "strings"

func main() {
	configFile := flag.String("config","","Path to the configuration file.")

	if *configFile == "" {
		mainDefault()
	} else {
		log.Fatal("Config file not implemented yet!")
	}
}

func readln(reader *bufio.Reader, msg string, args... interface{}) string {
	fmt.Printf(msg, args...)	

	line, err := reader.ReadString('\n')

	if err != nil {
		log.Fatalf("Reading secret failed: %v", err.Error())
	}

	return strings.Trim(line, "\r\t\n ")
}

func mainDefault() {
	reader := bufio.NewReader(os.Stdin)

	rootToken := readln(reader, "Root token [leave empty to generate new one]: ")

	tg, err := rndstring.NewStringGenerator("hex", 14)
	
	if err != nil {
		log.Fatal(err.Error())
	}

	if rootToken == "" {
		rootToken = tg.Generate()
	}

	log.Printf("Root Token: %s", rootToken)

	apiState := &ApiState{
		ContentTypes: map[string]string {
			".json" : "application/json",
			".txt" : "text/plain",
			".log" : "text/plain",
		},
		Delimiters: map[string][]byte {
			".log" : []byte("\n"),
		},
		DefaultContentType: "application/octet-stream",
		DataStore: NewMemDataStore(rootToken),
		StringGenerator: tg,
	}

	apiRouter := NewAPI(apiState)

	loggedRouter := handlers.RecoveryHandler()(handlers.LoggingHandler(os.Stdout, apiRouter))
	log.Fatal(http.ListenAndServe(":3000", loggedRouter))
}
