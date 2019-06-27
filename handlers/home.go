package handlers

import (
	"net/http"
)



func HomeHandler(w http.ResponseWriter,req *http.Request) {
	w.Write([]byte("Hello World!"))
}
