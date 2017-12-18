package main

import (
	"os"
	"os/signal"
	"syscall"

	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"maunium.net/go/lindeb/api"
	flag "maunium.net/go/mauflag"
)

var configPath = flag.MakeFull("c", "config", "Path to the config file.", "config.yaml").String()
var wantHelp, _ = flag.MakeHelpFlag()

func main() {
	flag.SetHelpTitles("lindeb - mau\\Lu Link Database", "lindeb [-c /path/to/config] [-h]")
	err := flag.Parse()
	if err != nil {
		fmt.Println(err)
		*wantHelp = true
	}
	if *wantHelp {
		flag.PrintHelp()
		os.Exit(1)
	}

	config, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Println("Failed to load config:", err)
		os.Exit(10)
	}

	db, err := config.Database.Connect()
	if err != nil {
		fmt.Println("Database connection failed:", err)
		os.Exit(11)
	}
	db.CreateTables()

	search, err := config.Elastic.Connect()
	if err != nil {
		fmt.Println("Elasticsearch connection failed:", err)
		os.Exit(12)
	}

	r := mux.NewRouter()

	apiObj := &api.API{
		DB:      db,
		Elastic: search,
	}
	apiObj.AddHandler(r.PathPrefix(config.API.Prefix).Subrouter())
	config.Frontend.AddHandler(r)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		if sig == os.Interrupt {
			// Interactive interrupt
			fmt.Print("\nShutting down...\n")
		}
		db.Close()
		// TODO check if this does what's expected
		search.Stop()
		os.Exit(1)
	}()

	err = config.API.ListenAndServe(r)
	if err != nil {
		fmt.Println("HTTP server quit unexpectedly:", err)
		os.Exit(20)
	}
}
