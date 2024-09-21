package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

type RequestResult struct {
	Duration   time.Duration
	StatusCode int
	Error      error
}

// Send POST requests to the leader
func sendPhraseToLeader(phrase string, leaderURL string, ch chan<- RequestResult, wg *sync.WaitGroup) {
	defer wg.Done()

	startTime := time.Now()

	data := url.Values{}
	data.Set("text", phrase)

	resp, err := http.PostForm(fmt.Sprintf("%s/post", leaderURL), data)
	duration := time.Since(startTime)

	if err != nil {
		ch <- RequestResult{
			Duration:   duration,
			StatusCode: 0, // Status 0 indicates an error (since no valid response was received)
			Error:      err,
		}
		return
	}

	defer resp.Body.Close()

	ch <- RequestResult{
		Duration:   duration,
		StatusCode: resp.StatusCode,
		Error:      nil,
	}
}

// Send GET requests to replicas to query word count
func queryReplica(word string, replicaURL string, ch chan<- RequestResult, wg *sync.WaitGroup) {
	defer wg.Done()

	startTime := time.Now()
	queryURL := fmt.Sprintf("%s/wordcount?word=%s", replicaURL, word)

	resp, err := http.Get(queryURL)
	duration := time.Since(startTime)

	if err != nil {
		ch <- RequestResult{
			Duration:   duration,
			StatusCode: 0,
			Error:      err,
		}
		return
	}
	defer resp.Body.Close()

	var result map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		ch <- RequestResult{
			Duration:   duration,
			StatusCode: 0,
			Error:      err,
		}
		return
	}

	ch <- RequestResult{
		Duration:   duration,
		StatusCode: resp.StatusCode,
		Error:      nil,
	}
}

func main() {
	// take the leader and replicas URLs from the command line
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <leaderURL> <replicaURL1> <replicaURL2> ...")

		return
	}

	leaderURL := os.Args[1]
	replicaURLs := os.Args[2:]

	// Phrases to send to the leader
	phrases := []string{
		"hello world", "world of go", "distributed systems", "hello distributed systems", "go is great",
		"concurrency in go", "parallel requests", "performance test", "high load test", "go concurrency model",
	}

	numRequests := 1000          // Number of parallel requests to leader/replicas
	wordToQuery := "distributed" // Word to query on replicas

	leaderCh := make(chan RequestResult, numRequests)
	replicaCh := make(chan RequestResult, numRequests)
	var wg sync.WaitGroup

	startTest := time.Now()

	// Fire parallel requests to the leader
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go sendPhraseToLeader(phrases[i%len(phrases)], leaderURL, leaderCh, &wg)
	}

	// Wait for all POST requests to finish
	wg.Wait()

	// Now simulate querying replicas in parallel to check word counts
	for i := 0; i < len(replicaURLs); i++ {
		for j := 0; j < numRequests/len(replicaURLs); j++ {
			wg.Add(1)
			go queryReplica(wordToQuery, replicaURLs[i], replicaCh, &wg)
		}
	}

	wg.Wait()
	close(leaderCh)
	close(replicaCh)

	// Process leader results
	successCountLeader := 0
	failureCountLeader := 0
	var totalDurationLeader time.Duration
	var slowestRequestLeader, fastestRequestLeader RequestResult

	for result := range leaderCh {
		if result.Error != nil || result.StatusCode != http.StatusAccepted {
			failureCountLeader++
		} else {
			successCountLeader++
		}

		totalDurationLeader += result.Duration

		// Track the slowest and fastest request
		if slowestRequestLeader.Duration < result.Duration {
			slowestRequestLeader = result
		}
		if fastestRequestLeader.Duration == 0 || fastestRequestLeader.Duration > result.Duration {
			fastestRequestLeader = result
		}
	}

	// Process replica results
	successCountReplica := 0
	failureCountReplica := 0
	var totalDurationReplica time.Duration
	var slowestRequestReplica, fastestRequestReplica RequestResult

	for result := range replicaCh {
		if result.Error != nil || result.StatusCode != http.StatusOK {
			fmt.Printf("last error: %s", result.Error)
			fmt.Printf("last error: %d", result.StatusCode)
			failureCountReplica++
		} else {
			successCountReplica++
		}

		totalDurationReplica += result.Duration

		// Track the slowest and fastest request
		if slowestRequestReplica.Duration < result.Duration {
			slowestRequestReplica = result
		}
		if fastestRequestReplica.Duration == 0 || fastestRequestReplica.Duration > result.Duration {
			fastestRequestReplica = result
		}
	}

	totalTestDuration := time.Since(startTest)

	// Output the results for the leader
	fmt.Println("=== Leader Performance ===")
	fmt.Printf("Total Requests to Leader: %d\n", numRequests)
	fmt.Printf("Successful Requests: %d\n", successCountLeader)
	fmt.Printf("Failed Requests: %d\n", failureCountLeader)
	fmt.Printf("Average Request Time: %v\n", totalDurationLeader/time.Duration(numRequests))
	fmt.Printf("Fastest Request Time: %v\n", fastestRequestLeader.Duration)
	fmt.Printf("Slowest Request Time: %v\n", slowestRequestLeader.Duration)

	// Output the results for the replicas
	fmt.Println("\n=== Replica Performance ===")
	fmt.Printf("Total Requests to Replicas: %d\n", numRequests)
	fmt.Printf("Successful Requests: %d\n", successCountReplica)
	fmt.Printf("Failed Requests: %d\n", failureCountReplica)
	fmt.Printf("Average Request Time: %v\n", totalDurationReplica/time.Duration(numRequests))
	fmt.Printf("Fastest Request Time: %v\n", fastestRequestReplica.Duration)
	fmt.Printf("Slowest Request Time: %v\n", slowestRequestReplica.Duration)

	// Overall test duration
	fmt.Printf("\nTotal Test Duration: %v\n", totalTestDuration)

	// Query the leader for a full sync
	leaderSyncURL := fmt.Sprintf("%s/sync", leaderURL)
	resp, err := http.Get(leaderSyncURL)
	if err != nil {
		fmt.Printf("Error querying leader for full sync: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Failed to get full sync from leader, status code: %d\n", resp.StatusCode)
		return
	}

	var leaderData map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&leaderData); err != nil {
		fmt.Printf("Error decoding leader full sync response: %v\n", err)
		return
	}

	// Query each replica for each word in the leader response and check for consistency
	inconsistencies := 0
	for word, leaderCount := range leaderData {
		for _, replicaURL := range replicaURLs {
			wg.Add(1)
			go func(word string, leaderCount int, replicaURL string) {
				defer wg.Done()

				queryURL := fmt.Sprintf("%s/wordcount?word=%s", replicaURL, word)
				resp, err := http.Get(queryURL)
				if err != nil {
					fmt.Printf("Error querying replica %s for word %s: %v\n", replicaURL, word, err)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					fmt.Printf("Failed to get word count from replica %s for word %s, status code: %d\n", replicaURL, word, resp.StatusCode)
					return
				}

				var replicaData map[string]int
				if err := json.NewDecoder(resp.Body).Decode(&replicaData); err != nil {
					fmt.Printf("Error decoding replica %s response for word %s: %v\n", replicaURL, word, err)
					return
				}

				if replicaCount, ok := replicaData[word]; !ok || replicaCount != leaderCount {
					fmt.Printf("Inconsistency found for word %s: leader count = %d, replica %s count = %d\n", word, leaderCount, replicaURL, replicaCount)
					inconsistencies++
				}
			}(word, leaderCount, replicaURL)
		}
	}

	wg.Wait()

	if inconsistencies == 0 {
		fmt.Println("All replicas are consistent with the leader.")
	} else {
		fmt.Printf("Found %d inconsistencies between leader and replicas.\n", inconsistencies)
	}

	fmt.Printf("\nTest completed\n")
}
