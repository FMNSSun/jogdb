package main

import . "github.com/FMNSSun/jogdb"
import "github.com/FMNSSun/libtoken"
import "net/http"
import "github.com/gorilla/handlers"
import "os"
import "log"

func main() {
	tg, err := libtoken.NewTokenGenerator("hex", 14)
	
	if err != nil {
		log.Fatal(err.Error())
	}

	rootToken := tg.Generate()
	rootToken = "dbd489bd16926ceafda8512c23cb"
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
		TokenGenerator: tg,
	}

	apiRouter := NewAPI(apiState)

	loggedRouter := handlers.LoggingHandler(os.Stdout, apiRouter)
	log.Fatal(http.ListenAndServe(":3000", loggedRouter))
}
