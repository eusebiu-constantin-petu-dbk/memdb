package system_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"memdb/pkg/db"
	"memdb/pkg/server"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	leaderPort    = "9090"
	leaderAddress = fmt.Sprintf("http://localhost:%s", leaderPort)

	replicasPort = []string{"9091", "9092", "9093"}
)

// Start the server in a separate goroutine
func startServers() func() {
	tmpDir, err := os.CreateTemp("/tmp", "memdb-*")
	if err != nil {
		panic(err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	leaderDB := db.NewLeader(tmpDir.Name(), logger)

	leader := server.NewLeaderServer(leaderDB, leaderPort, logger)
	go func() {
		leader.RunServer()
	}()

	time.Sleep(1 * time.Second)

	replicas := []*server.ReplicaServer{}
	for _, port := range replicasPort {
		replicaDB := db.NewReplica(logger)
		replicaServer := server.NewReplicaServer(replicaDB, port, leaderAddress, logger)

		replicas = append(replicas, replicaServer)

		leader.AddReplica(fmt.Sprintf("http://localhost:%s", port))

		go func() {
			replicaServer.RunServer()
		}()
	}

	return func() {
		leader.Shutdown(context.Background())
		for _, replica := range replicas {
			replica.Shutdown(context.Background())
		}
		// cleanup
		os.RemoveAll(tmpDir.Name())
	}
}

// Test suite for system testing without mocking
func TestSystem(t *testing.T) {
	// Start the leader server in a separate goroutine
	cleanup := startServers()

	defer cleanup()

	// Give the server some time to start
	time.Sleep(1 * time.Second)

	phrases := []string{
		"hello world",
		"world of go",
		"distributed systems are awesome",
		"hello distributed world",
	}

	// Send multiple phrases using the sendPhrase function
	for _, phrase := range phrases {
		err := sendPhrase(phrase)
		if err != nil {
			t.Fatalf("failed to send phrase: %v", err)
		}
	}

	time.Sleep(2 * time.Second) // wait for replicating b/c of eventual consistency

	Convey("When phrases are sent to the server", t, func() {
		Convey("The word count for 'hello' should be correct, leader", func() {
			db, err := getLeaderDatabse()
			So(err, ShouldBeNil)

			So(db["hello"], ShouldEqual, 2)
			So(db["world"], ShouldEqual, 3)
			So(db["distributed"], ShouldEqual, 2)
			So(db["go"], ShouldEqual, 1)
		})

		// wait for replicating b/c of eventual consistency
		// time.Sleep(1 * time.Second)
		Convey("The word count for 'hello' should be correct", func() {
			for _, replicaPort := range replicasPort {
				count, err := readWordCount("hello", replicaPort)
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 2)
			}
		})

		Convey("The word count for 'world' should be correct", func() {
			for _, replicaPort := range replicasPort {
				count, err := readWordCount("world", replicaPort)
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 3)
			}
		})

		Convey("The word count for 'distributed' should be correct", func() {
			for _, replicaPort := range replicasPort {
				count, err := readWordCount("distributed", replicaPort)
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 2)
			}
		})

		Convey("The word count for 'go' should be correct", func() {
			for _, replicaPort := range replicasPort {
				count, err := readWordCount("go", replicaPort)
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 1)
			}
		})

		Convey("The word count for 'systems' should be correct", func() {
			for _, replicaPort := range replicasPort {
				count, err := readWordCount("systems", replicaPort)
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 1)
			}
		})

		Convey("The word count for 'awesome' should be correct", func() {
			for _, replicaPort := range replicasPort {
				count, err := readWordCount("awesome", replicaPort)
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 1)
			}
		})

		Convey("The word count for 'of' should be correct", func() {
			for _, replicaPort := range replicasPort {
				count, err := readWordCount("of", replicaPort)
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 1)
			}
		})
	})
}

func sendPhrase(phrase string) error {
	data := url.Values{}
	data.Set("text", phrase)

	resp, err := http.PostForm(fmt.Sprintf("%s/post", leaderAddress), data)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("failed to send phrase, status code: %d", resp.StatusCode)
	}

	return nil
}

func readWordCount(word string, replicaPort string) (int, error) {
	queryURL := fmt.Sprintf("http://localhost:%s/wordcount?word=%s", replicaPort, word)

	resp, err := http.Get(queryURL)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get word count, status code: %d", resp.StatusCode)
	}

	var result map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	count, ok := result[word]
	if !ok {
		return 0, fmt.Errorf("invalid response, 'count' field missing")
	}

	return count, nil
}

func getLeaderDatabse() (map[string]int, error) {
	queryURL := fmt.Sprintf("%s/sync", leaderAddress)

	resp, err := http.Get(queryURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get word count, status code: %d", resp.StatusCode)
	}

	var result map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
