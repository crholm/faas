package routes

import "math/rand"

type LoadBalancer interface {
	Next(f *Function, do func(route *Route))
}


type RandomBalancer struct {
	
}


func (l *RandomBalancer) Next(f *Function, do func(route *Route)){
	paths := f.Paths()
	i := rand.Intn(len(paths))
	do(f.Get(paths[i]))
}