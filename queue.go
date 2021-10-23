package main

import (
	"fmt"
	"math"
	"math/rand"
)

const epsilon = 1e-6

func formatTime(t int) string {
	h := t / 60
	m := t % 60
	return fmt.Sprintf("%02d:%02d", h, m)
}

type Poisson struct {
	lambda float64
	maxn   int
	p      []float64
	rng    *rand.Rand
}

func NewPoisson(lambda float64, maxn int, seed int64) *Poisson {
	p := make([]float64, maxn+1)
	p[0] = math.Exp(-lambda)
	for i := 1; i <= maxn; i++ {
		p[i] = p[i-1] * lambda / float64(i)
	}
	return &Poisson{
		lambda: lambda,
		maxn:   maxn,
		p:      p,
		rng:    rand.New(rand.NewSource(seed)),
	}
}

func (p *Poisson) Get() int {
	x := p.rng.Float64()
	cum := float64(0)
	for i, pi := range p.p {
		cum += pi
		if x <= cum {
			return i
		}
	}
	return p.maxn
}

type Exponential struct {
	lambda float64
	rng    *rand.Rand
}

func NewExponential(lambda float64, seed int64) *Exponential {
	return &Exponential{
		lambda: lambda,
		rng:    rand.New(rand.NewSource(seed)),
	}
}

func (e *Exponential) Get() float64 {
	r := e.rng.Float64()
	L := float64(0)
	R := float64(1e100)
	for R-L > epsilon {
		mid := (L + R) * 0.5
		p := float64(1) - math.Exp(-e.lambda*mid)
		if p < r {
			L = mid
		} else {
			R = mid
		}
	}
	return L
}

type Customer struct {
	ArrivalTime, ServedTime, FinishTime int
	Server                              int
}

func (c *Customer) WaitTime() int {
	return c.ServedTime - c.ArrivalTime
}

func (c *Customer) ServiceTime() int {
	return c.FinishTime - c.ServedTime
}

func (c *Customer) SpentTime() int {
	return c.FinishTime - c.ArrivalTime
}

type Simulation struct {
	nServers                 int
	startTime, endTime       int
	customerRate, serverRate float64

	customerDist *Poisson
	serverDist   []*Exponential
}

func NewSimulation(startTime, endTime, nServers int, customerRate, serverRate float64, seed int64) *Simulation {
	poisson := NewPoisson(customerRate/60, 100, seed)
	erng := rand.New(rand.NewSource(seed))
	exp := make([]*Exponential, nServers)
	for i := range exp {
		exp[i] = NewExponential(float64(1)/(float64(60)/serverRate), erng.Int63())
	}

	return &Simulation{
		nServers:     nServers,
		startTime:    startTime,
		endTime:      endTime,
		customerRate: customerRate,
		serverRate:   serverRate,
		customerDist: poisson,
		serverDist:   exp,
	}
}

type SimulationResult struct {
	TotalTime          int
	TotalCustomers     int
	TotalServers       int
	AverageWaitTime    float64
	AverageServiceTime float64
}

func (s *Simulation) Simulate(verbose bool) SimulationResult {
	customerIndex := 0
	totalWaitTime := 0
	totalServiceTime := 0
	serversLastIdleTime := make([]int, s.nServers)
	for t := s.startTime; t < s.endTime; t++ {
		k := s.customerDist.Get()
		for ik := 0; ik < k; ik++ {
			customerIndex++
			c := Customer{ArrivalTime: t}
			// find the earliest available server
			timeServed, serverIndex := -1, -1
			for j, t := range serversLastIdleTime {
				availableTime := t
				if t < c.ArrivalTime {
					availableTime = c.ArrivalTime
				}
				if availableTime < timeServed || serverIndex == -1 {
					timeServed, serverIndex = availableTime, j
					if timeServed == 0 {
						break
					}
				}
			}

			serviceTime := int(math.Round(s.serverDist[serverIndex].Get()))
			c.Server = serverIndex
			c.ServedTime = timeServed
			c.FinishTime = timeServed + serviceTime
			serversLastIdleTime[serverIndex] = c.FinishTime

			totalWaitTime += c.WaitTime()
			totalServiceTime += serviceTime

			if verbose {
				fmt.Printf("Customer %d:\n", customerIndex)
				fmt.Printf("\tArrival   : %s\n", formatTime(c.ArrivalTime))
				fmt.Printf("\tServedTime: %s (by server %d) (WaitTime = %d minutes)\n", formatTime(c.ServedTime), c.Server, c.WaitTime())
				fmt.Printf("\tFinishTime: %s (ServiceTime = %d minutes)\n", formatTime(c.FinishTime), serviceTime)
			}
		}
	}

	return SimulationResult{
		TotalTime:          s.endTime - s.startTime,
		TotalCustomers:     customerIndex,
		TotalServers:       s.nServers,
		AverageWaitTime:    float64(totalWaitTime) / float64(customerIndex),
		AverageServiceTime: float64(totalServiceTime) / float64(customerIndex),
	}
}

func simulateOnce(seed int64) {
	startTime := 8 * 60 // 08:00
	endTime := 16 * 60  // 16:00
	customerRate := 5.8 // 5.8 customers per hour
	serverRate := 6.0   // 6 customers per hour, or 10 minutes per customer
	nServers := 2

	s := NewSimulation(startTime, endTime, nServers, customerRate, serverRate, seed)
	result := s.Simulate(true)

	fmt.Println()
	fmt.Printf("Simulation Time    : %d hours\n", result.TotalTime/60)
	fmt.Printf("Total Customers    : %d (%.6f customers/hour)\n", result.TotalCustomers, float64(result.TotalCustomers)/(float64(result.TotalTime)/float64(60)))
	fmt.Printf("Total Servers      : %d\n", result.TotalServers)
	fmt.Printf("Average WaitTime   : %.6f minutes\n", result.AverageWaitTime)
	fmt.Printf("Average ServiceTime: %.6f minutes\n", result.AverageServiceTime)
}

func simulateGrid(seed int64) {
	rng := rand.New(rand.NewSource(seed))

	times := []int{1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000, 20000, 50000, 100000, 200000, 500000, 1000000}
	nServers := []int{1, 2}
	customerRate := 5.8 // 5.8 customers per hour
	serverRate := 6.0   // 6 customers per hour, or 10 minutes per customer

	fmt.Println("total_time,total_servers,total_customers,customer_rate,server_rate,actual_customer_rate,actual_server_rate,average_wait_time")

	for _, t := range times {
		for _, ns := range nServers {
			// We run the simulation several times for better convergence
			n := 1000 / t
			if n == 0 {
				n = 1
			}
			result := SimulationResult{}
			for i := 0; i < n; i++ {
				s := NewSimulation(0, t*60, ns, customerRate, serverRate, rng.Int63())
				r := s.Simulate(false)
				if r.TotalCustomers > 0 {
					result.TotalCustomers += r.TotalCustomers
					result.AverageWaitTime += r.AverageWaitTime
					result.AverageServiceTime += r.AverageServiceTime
				}
			}
			result.TotalTime = t * 60
			result.TotalServers = ns
			result.TotalCustomers /= n
			result.AverageWaitTime /= float64(n)
			result.AverageServiceTime /= float64(n)

			fmt.Printf("%d,%d,%d,%.4f,%.4f,%.4f,%.4f,%.4f\n", result.TotalTime/60, result.TotalServers, result.TotalCustomers, customerRate, serverRate, float64(result.TotalCustomers)/(float64(result.TotalTime)/60), float64(60)/result.AverageServiceTime, result.AverageWaitTime)
		}
	}
}

func main() {
	seed := int64(2021)

	// simulateOnce(seed)
	simulateGrid(seed)
}
