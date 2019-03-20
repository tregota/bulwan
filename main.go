package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// global vars
var serverEndpoint *Endpoint
var username string
var port int
var privatekey string
var tunnelActive bool
var sshConn *SSHConn
var router *mux.Router
var onclose string

// functions
func proxy(url string) http.HandlerFunc {

	return func(wr http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		req, err := http.NewRequest(r.Method, url+r.URL.String(), r.Body)
		if err != nil {
			http.Error(wr, err.Error(), http.StatusInternalServerError)
			return
		}
		for name, value := range r.Header {
			req.Header.Set(name, value[0])
		}

		client := &http.Client{
			Timeout: time.Second * 60,
		}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(wr, err.Error(), http.StatusInternalServerError)
			return
		}

		for k, v := range resp.Header {
			wr.Header().Set(k, v[0])
		}
		wr.WriteHeader(resp.StatusCode)
		io.Copy(wr, resp.Body)
		resp.Body.Close()
	}
}

func remotePort() (err error) {

	if !tunnelActive {
		fmt.Printf("Tunnel inactive, waiting..\n")
		for !tunnelActive {
			time.Sleep(time.Second)
		}
	}

	sshConn, err = serverEndpoint.SSHDial(username, privatekey)
	if err != nil {
		return fmt.Errorf("%s:%d remotePort sshdial - %v", serverEndpoint.Host, serverEndpoint.Port, err)
	}
	defer sshConn.Connection.Close()

	// blocking call
	listener, err := sshConn.ReverseTunnelForceListen(port, username)
	if err != nil {
		if strings.Contains(err.Error(), "unable to bind port") {
			//well crap..
			os.Exit(1) // for now
		}
		return fmt.Errorf("%s:%d remotePort listen - %v", sshConn.Server.Host, port, err)
	}
	defer listener.Close()

	srv := &http.Server{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		Handler:      router,
	}
	defer srv.Close()
	return srv.Serve(listener)
}

func localPort() error {

	srv := &http.Server{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		Handler:      router,
		Addr:         ":35300",
	}
	defer srv.Close()
	return srv.ListenAndServe()
}

func openTunnel(w http.ResponseWriter, r *http.Request) {
	if !tunnelActive {
		fmt.Printf("Opening tunnel..\n")
		flagFile, err := os.Create("tunnelactive.flag")
		if err != nil {
			fmt.Printf("openTunnel error: %s\n", err)
		} else {
			flagFile.Close()
		}
		tunnelActive = true
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func closeTunnel(w http.ResponseWriter, r *http.Request) {
	if tunnelActive {
		fmt.Printf("Closing tunnel..\n")
		err := os.Remove("tunnelactive.flag")
		if err != nil {
			fmt.Printf("closeTunnel error: %s\n", err)
		}
		go onCloseTunnel()
		tunnelActive = false
		sshConn.Connection.Close()
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func onCloseTunnel() {
	if len(onclose) > 0 {

		// sadly we don't seem to have any root certificates available (certificate signed by unknown authority)
		client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
		req, _ := http.NewRequest(http.MethodGet, onclose, nil)

		_, err := client.Do(req)
		if err != nil {
			fmt.Printf("onCloseTunnel error: %s\n", err)
		}
	}
}

func main() {
	fmt.Printf("Starting..\n")
	defer fmt.Printf("Exiting..\n")

	if _, err := os.Stat("tunnelactive.flag"); os.IsNotExist(err) {
		tunnelActive = false
	} else {
		tunnelActive = true
	}

	serverport, err := strconv.Atoi(os.Getenv("SERVER_PORT"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	serverEndpoint = &Endpoint{
		Host:          os.Getenv("SERVER_HOST"),
		Port:          serverport,
		PublicKey:     os.Getenv("SERVER_PUBLIC_KEY"), // take the relevant key from the known_hosts file
		PublicKeyType: os.Getenv("SERVER_PUBLIC_KEY_TYPE"),
	}
	username = os.Getenv("SSH_USERNAME")
	if len(username) == 0 {
		fmt.Fprintf(os.Stderr, "error: env SSH_USERNAME empty\n")
		os.Exit(1)
	}
	port, err = strconv.Atoi(os.Getenv("SSH_LISTEN_PORT"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	privatekey = os.Getenv("SSH_PRIVATE_KEY")
	if len(privatekey) == 0 {
		fmt.Fprintf(os.Stderr, "error: env SSH_PRIVATE_KEY empty\n")
		os.Exit(1)
	}
	onclose = os.Getenv("HTTP_GET_ON_CLOSE")

	router = mux.NewRouter()
	router.HandleFunc("/open", openTunnel).Methods("GET")
	router.HandleFunc("/close", closeTunnel).Methods("GET")

	num := 1
	for true {
		prefix := os.Getenv(fmt.Sprintf("EXPOSED_HTTPSERVER_PREFIX_%d", num))
		url := os.Getenv(fmt.Sprintf("EXPOSED_HTTPSERVER_URL_%d", num))

		if len(url) > 0 {
			router.PathPrefix(fmt.Sprintf("/%s/", prefix)).Handler(http.StripPrefix(fmt.Sprintf("/%s", prefix), proxy(url)))
		} else {
			break
		}
		num++
	}

	go KeepAlive(localPort)
	err = KeepAlive(remotePort)
	if err != nil {
		fmt.Printf("remotePort Error: %s\n", err)
	}
}
