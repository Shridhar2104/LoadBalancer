package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Server interface {
	Address() string
	isAlive() bool
	Serve(w http.ResponseWriter, r *http.Request)
}

type simpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func newSimpleServer(addr string) *simpleServer {
	serverUrl, err := url.Parse(addr)
	handleErr(err)
	return &simpleServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

type LoadBalancer struct {
	port           string
	roundRobinCnt  int
	servers        []Server
}

func newLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port:           port,
		roundRobinCnt:  0,
		servers:        servers,
	}
}

func (s *simpleServer) Address() string { return s.addr }
func (s *simpleServer) isAlive() bool   { return true }

func (s *simpleServer) Serve(w http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(w, r)
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	server := lb.servers[lb.roundRobinCnt%len(lb.servers)]
	for !server.isAlive() {
		lb.roundRobinCnt++
		server = lb.servers[lb.roundRobinCnt%len(lb.servers)]
	}
	lb.roundRobinCnt++
	return server
}

func (lb *LoadBalancer) serverProxy(w http.ResponseWriter, r *http.Request) {
	targetServer := lb.getNextAvailableServer()
	fmt.Printf("forwarding request to target server %q\n", targetServer.Address())
	targetServer.Serve(w, r)
}

func main() {
	servers := []Server{
		newSimpleServer("https://www.facebook.com"),
		newSimpleServer("https://www.duckduckgo.com"),
		newSimpleServer("https://www.bing.com"),
	}
	lb := newLoadBalancer("8080", servers)

	handleRedirect := func(w http.ResponseWriter, r *http.Request) {
		lb.serverProxy(w, r)
	}

	http.HandleFunc("/", handleRedirect)

	fmt.Printf("server started at %s\n", lb.port)

	http.ListenAndServe(":"+lb.port, nil)
}
