package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var primes []int

func init() {
	log.Println(fmt.Sprintf("Computing prime numbers between 0 - %v", math.MaxInt16))
	primes = compute(math.MaxInt16)
	log.Println(fmt.Sprintf("Between 0 - %v, there are %v prime numbers.", math.MaxInt16, len(primes)))
	log.Println(fmt.Sprintf("Largest prime number with 0 - %v is %v", math.MaxInt16, primes[len(primes)-1]))
}

func primeHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	computeQryParam := query.Get("compute")
	if computeQryParam == "" {
		http.Error(w, "Please provide the nth value via '?compute=' query parameter.",
			http.StatusBadRequest)
		return
	}

	pos, err := strconv.Atoi(computeQryParam)
	if err != nil {
		http.Error(w, fmt.Sprintf("Please provide a valid nth value. Invalid input %v", computeQryParam),
			http.StatusBadRequest)
		return
	}

	primeNumber, found := nth(pos)
	if !found {
		http.Error(w, fmt.Sprintf("Failed to find the %vth prime number. There are only %v prime numbers between 0 - %v", pos, len(primes), math.MaxInt16),
			http.StatusInternalServerError)
		return
	}
	_, _ = w.Write([]byte(fmt.Sprintf("The %vth Prime number is %v\n", pos, primeNumber)))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", primeHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/readiness", readinessHandler)

	srv := &http.Server{
		Handler:      mux,
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start Server
	go func() {
		log.Println("Starting Server")
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	// Graceful Shutdown
	waitForShutdown(srv)
}

// Nth determines the nth prime number
func nth(n int) (int, bool) {
	if n <= 0 {
		return 0, false
	}
	if n > len(primes) {
		return 0, false
	}
	return primes[n-1], true
}

func waitForShutdown(srv *http.Server) {
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	<-interruptChan

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	srv.Shutdown(ctx)

	log.Println("Shutting down")
	os.Exit(0)
}

// Computes the prime numbers
// Eliminating composites by prime divisors, the Sieve of Eratosthenes
func compute(limit int) []int {
	segmentSize := int(math.Sqrt(float64(limit)))

	// Generation of small primes
	notPrime := make([]bool, segmentSize+1)
	for i := 0; i*i < segmentSize && i < 2; i++ {
		notPrime[i] = true
	}

	for i := 2; i*i < segmentSize; i++ {
		if !notPrime[i] {
			for j := i * i; j <= segmentSize; j += i {
				notPrime[j] = true
			}
		}
	}

	sieve := make([]bool, segmentSize)
	var allPrimes []int
	for i := 2; i < len(notPrime); i++ {
		if !notPrime[i] {
			allPrimes = append(allPrimes, i)
		}
	}

	var next []int
	s := 3
	n := segmentSize + 1
	if n%2 == 0 {
		n--
	}

	for low := 0; low <= limit; low += segmentSize {
		for i := 0; i < segmentSize; i++ {
			sieve[i] = true
		}

		// Current segment on the interval [low, high]
		high := low + segmentSize - 1
		if high > limit {
			high = limit
		}

		// Add new sieving primes <= √high
		for ; s*s <= high; s += 2 {
			if !notPrime[s] {
				next = append(next, s*s-low)
			}
		}

		// Sieve the current segment
		for i := 0; i < len(next); i++ {
			j := next[i]
			for k := allPrimes[i+1] * 2; j < segmentSize; j += k {
				sieve[j] = false
			}

			next[i] = j - segmentSize
		}

		// Skip the first set of primes because allPrimes already has them
		if low > 0 {
			for ; n <= high; n += 2 {
				if sieve[n-low] {
					allPrimes = append(allPrimes, n)
				}
			}
		}
	}
	return allPrimes
}