package server

import (
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
)

type server struct {
	storage map[string]string
}

func NewServer() *server {
	s := &server{storage: make(map[string]string)}
	return s
}

func getKey(r *http.Request) string {
	vars := mux.Vars(r)
	return vars["key"]
}

func (s *server) pingHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s!", r.URL.Path[1:])
	w.Write([]byte("pong!"))
}

func (s *server) getHandler(w http.ResponseWriter, r *http.Request) {
	key := getKey(r)
	log.Printf("get key %s", key)

	value, ok := s.storage[key]
	if !ok {
		log.Printf("get key %s failed: key not found", key)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("key %s not found", key)))
		return
	}

	w.Write([]byte(value))
}

func (s *server) postHandler(w http.ResponseWriter, r *http.Request) {
	key := getKey(r)
	log.Printf("set key %s", key)

	defer r.Body.Close()
	value, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("set key %s failed: %s", key, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("server error")))
		return
	}

	s.storage[key] = string(value)

	w.Write(nil)
}

func (s *server) putHandler(w http.ResponseWriter, r *http.Request) {
	key := getKey(r)
	log.Printf("insert key %s", key)

	defer r.Body.Close()
	value, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("insert key %s failed: %s", key, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("server error")))
		return
	}

	if _, ok := s.storage[key]; ok {
		log.Printf("insert key %s failed: key exists", key)
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(fmt.Sprintf("key %s exists", key)))
		return
	}

	s.storage[key] = string(value)

	w.Write(nil)
}

func (s *server) deleteHandler(w http.ResponseWriter, r *http.Request) {
	key := getKey(r)
	log.Printf("delete key %s", key)

	_, ok := s.storage[key]
	if !ok {
		log.Printf("delete key %s failed: key not found", key)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("key %s not found", key)))
		return
	}

	delete(s.storage, key)

	w.Write(nil)
}

func (s *server) Start(port string) {
	r := mux.NewRouter()
	r.HandleFunc("/ping", s.pingHandler)
	r.HandleFunc("/key/{key}", s.getHandler).Methods(http.MethodGet)
	r.HandleFunc("/key/{key}", s.postHandler).Methods(http.MethodPost)
	r.HandleFunc("/key/{key}", s.putHandler).Methods(http.MethodPut)
	r.HandleFunc("/key/{key}", s.deleteHandler).Methods(http.MethodDelete)
	log.Println("start server on port: " + port)
	//http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
