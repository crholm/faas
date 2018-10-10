package routes

import (
	"time"
	"fmt"
	"github.com/docker/docker/client"
	"context"
	"github.com/docker/docker/api/types"
	"strings"
	"strconv"
	"net/http/httputil"
	"net/url"
	"net/http"
)




var pollInterval = time.Second * 10

func New(labelPrefix string) *Routing{
	return &Routing{
		labelPrefix:  labelPrefix,
		functions:    make(map[string]*Function),
		loadBalancer: &RandomBalancer{},
	}
}


type Routing struct{
	stopped      bool
	labelPrefix  string
	loadBalancer LoadBalancer
	functions    map[string]*Function
}


func (r *Routing) Start(){
	fmt.Println("Staring routing discovery")

	cli, err := client.NewClientWithOpts(client.WithVersion("1.38"))
	if err != nil {
		panic(err)
	}

	go func() {
		for{
			if r.stopped {
				fmt.Println("Souting discovery stopped")
				return
			}
			r.refresh(cli)
			time.Sleep(pollInterval)
		}
	}()



	go func() {



		msg, errs := cli.Events(context.Background(), types.EventsOptions{})

		fmt.Println("Listening to events")
		for{
			select {
			case m := <-msg:
				fmt.Println("EVENT", m.Action)
				r.refresh(cli)
			case err = <-errs:
				fmt.Println("err", err)
			}
		}



	}()

}



func (r *Routing) StartWith(balancer LoadBalancer){
	if balancer != nil{
		r.loadBalancer = balancer
	}

	r.Start()
}

func (r *Routing) Stop(){
	r.stopped = true
}


func (r *Routing) refresh(cli *client.Client){



	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	var services []types.Container
	for _, container := range containers {
		for k, _ := range container.Labels{
			if strings.HasPrefix(k, r.labelPrefix){
				services = append(services, container)
				break
			}
		}
	}

	fmt.Println(len(services))
	before := time.Now()
	for _, container := range services {
		r.parsContainer(container)
	}
	r.clean(before)


	for name, fun := range r.functions{
		fmt.Println(name, ":")
		fmt.Println(" - ", fun.name)
		for k, rr := range fun.routes{
			fmt.Println("   - ", k, "\t",rr.lastUpdated )
		}

	}

}

func (r *Routing) clean(before time.Time){

	for name, function := range r.functions{
		for key, route := range function.routes{
			if route.lastUpdated.Before(before){
				delete(function.routes, key)
				fmt.Println("Removing dead enpoint ", name, key)
			}
		}

		if len(function.routes) == 0 {
			delete(r.functions, name)
		}

	}

}


func (r *Routing) parsContainer(container types.Container){

	name := container.Labels[r.labelPrefix + "name"]
	network := container.Labels[r.labelPrefix + "network"]
	port := container.Labels[r.labelPrefix + "port"]
	var ip string



	if port == "" && len(container.Ports) > 0{
		port = strconv.Itoa(int(container.Ports[0].PrivatePort))
	}
	if port == ""{
		port = "8080"
	}


	if len(container.NetworkSettings.Networks) == 0 {
		return
	}


	if network != "" {
		v, ok := container.NetworkSettings.Networks[network]
		if ok {
			ip = v.IPAddress
		}
	}


	if ip == ""  {
		for k, v := range  container.NetworkSettings.Networks{
			network = k
			ip = v.IPAddress
			break
		}
	}

	f, ok := r.functions[name]
	if !ok {
		f = &Function{
			name: name,
			routes: make(map[string]*Route),
		}
		r.functions[name] = f
	}

	path := fmt.Sprintf("%s:%s", ip, port)

	rr, ok := f.routes[path]
	if !ok {

		u, err := url.Parse(fmt.Sprintf("http://%s", path))
		if err != nil{
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(u)
		//TODO add error handler to retry and remove if there is not Route to host

		rr = &Route{
			ip: ip,
			port: port,
			proxy: proxy,

		}
		f.routes[path] = rr
	}

	rr.lastUpdated = time.Now()

}



func (r *Routing) Get(functionName string) *Function{
	return r.functions[functionName]
}


type Function struct{
	name string
	routes map[string]*Route
}

func (f *Function) Name() string{
	return f.name
}

type Route struct {
	ip string
	port string
	lastUpdated time.Time
	proxy *httputil.ReverseProxy
}

func (f *Function) Paths() ([]string){
	var paths []string
	for k := range f.routes {
		paths = append(paths, k)
	}
	return paths
}


func (f *Function) Get(path string) (*Route){
	return f.routes[path]
}




func writeError(w http.ResponseWriter, code int, message string){
	w.WriteHeader(code)
	fmt.Fprintf(w, message)
}


func (r *Routing) Proxy(functionName string, w http.ResponseWriter, req *http.Request){

	function := r.Get(functionName)

	if function == nil{
		writeError(w, http.StatusBadRequest, "could not find function")
		return
	}

	if len(function.routes) == 0{
		writeError(w, http.StatusInternalServerError, "could not find any routes for function")
		return
	}

	r.loadBalancer.Next(function, func(route *Route) {
		route.proxy.ServeHTTP(w, req)
	})

	return
}

