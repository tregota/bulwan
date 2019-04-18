package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// global vars
var settings = &settingstype{}
var tunnelActive bool
var sshConn *SSHConn
var router *mux.Router

// structs
type exposedHTTPServer struct {
	Prefix string
	URL    string
}
type settingstype struct {
	ServerHost          string
	ServerPort          int
	ServerPublicKey     string // take the relevant key from the known_hosts file
	ServerPublicKeyType string
	SSHUsername         string
	SSHListenPort       int
	SSHPrivateKey       string
	LocalServerAddr     string
	HTTPGetOnClose      string
	ExposedHTTPServers  []exposedHTTPServer
}

// functions
func proxy(url string) http.HandlerFunc {

	return func(wr http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		// fmt.Printf("Debug: request - %s\n", url+r.URL.String())

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
		fmt.Printf("Info: Tunnel inactive, waiting..\n")
		for !tunnelActive {
			time.Sleep(time.Second)
		}
	}

	serverEndpoint := Endpoint{
		Host:          settings.ServerHost,
		Port:          settings.ServerPort,
		PublicKey:     settings.ServerPublicKey,
		PublicKeyType: settings.ServerPublicKeyType,
	}

	sshConn, err = serverEndpoint.SSHDial(settings.SSHUsername, settings.SSHPrivateKey)
	if err != nil {
		return fmt.Errorf("%s:%d remotePort sshdial - %v", serverEndpoint.Host, serverEndpoint.Port, err)
	}
	defer sshConn.Connection.Close()

	// send a package to server every 2 minutes to keep session alive and notice broken connections
	testloop := sshConn.TestConnectionLoop(2*time.Minute, 10*time.Second)
	defer testloop.Stop()

	fmt.Printf("Info: ReverseTunnelForceListen on remote port %d\n", settings.SSHListenPort)
	listener, err := sshConn.ReverseTunnelForceListen(settings.SSHListenPort, settings.SSHUsername)
	if err != nil {
		if strings.Contains(err.Error(), "unable to bind port") {
			fmt.Printf("Fatal Error: ReverseTunnelForceListen - %s\n", err.Error())
			os.Exit(1) // for now
		}
		return fmt.Errorf("%s:%d remotePort listen - %v", sshConn.Server.Host, settings.SSHListenPort, err)
	}
	defer listener.Close()

	srv := &http.Server{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		Handler:      router,
	}
	defer srv.Close()
	fmt.Printf("Info: Starting server on remote port %d listener\n", settings.SSHListenPort)
	return srv.Serve(listener)
}

func localPort() error {

	srv := &http.Server{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		Handler:      router,
		Addr:         settings.LocalServerAddr,
	}
	defer srv.Close()
	fmt.Printf("Info: ListenAndServe on local port %s\n", settings.LocalServerAddr)
	return srv.ListenAndServe()
}

func openTunnel(w http.ResponseWriter, r *http.Request) {
	if !tunnelActive {
		fmt.Printf("Info: Opening tunnel..\n")
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
		fmt.Printf("Info: Closing tunnel..\n")
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
	if len(settings.HTTPGetOnClose) > 0 {

		// sadly we don't seem to have any root certificates available (certificate signed by unknown authority)
		client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
		req, _ := http.NewRequest(http.MethodGet, settings.HTTPGetOnClose, nil)

		_, err := client.Do(req)
		if err != nil {
			fmt.Printf("onCloseTunnel error: %s\n", err)
		}
	}
}

func main() {
	fmt.Printf("Info: Starting..\n")
	defer fmt.Printf("Info: Exiting..\n")

	if _, err := os.Stat("tunnelactive.flag"); os.IsNotExist(err) {
		tunnelActive = false
	} else {
		tunnelActive = true
	}

	err := LoadSettings(settings)
	if err != nil {
		fmt.Printf("Error: GetSettings - %s\n", err.Error())
		os.Exit(1)
	}

	router = mux.NewRouter()
	router.HandleFunc("/open", openTunnel).Methods("GET")
	router.HandleFunc("/close", closeTunnel).Methods("GET")

	for _, exposedHTTPServer := range settings.ExposedHTTPServers {
		fmt.Printf("Info: proxy - /%s/ to %s\n", exposedHTTPServer.Prefix, exposedHTTPServer.URL)
		router.PathPrefix(fmt.Sprintf("/%s/", exposedHTTPServer.Prefix)).
			Handler(http.StripPrefix(fmt.Sprintf("/%s", exposedHTTPServer.Prefix), proxy(exposedHTTPServer.URL)))
	}

	go KeepAlive(localPort)
	err = KeepAlive(remotePort)
	if err != nil {
		fmt.Printf("Error: remotePort - %s\n", err)
	}
}
