// ref: https://gist.github.com/enricofoltran/10b4a980cd07cb02836f70a4ab3e72d7
// ref: https://medium.com/@matryer/how-i-write-go-http-services-after-seven-years-37c208122831
package main

import (
	"Go_simple_webapp/server"
	"flag"
	"log"
	"os"
)


func main() {
	listenAddr := flag.String("l", ":5000", "server listen address")
	flag.Parse()

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)

	s := server.Server{
		Logger: logger,
	}
	s.ListenAndServe(*listenAddr)
}
