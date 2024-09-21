## memdb - POC for an in-memory db

`memdb` is a scalable in-memory database for word counting designed for high availability, partition tolerance, and eventual consistency.

It employs a leader-follower architecture where writes are directed to the leader and reads are served by replicas and the nodes communicates between them through REST API.


## Features

- **Scalable**: Easily scale out by adding more replicas.
- **High Availability**: Ensures data is available even in the event of node failures. If leader goes down, then writes are temporary unavailable but reads will work, if one replica goes down then the rest should work.
- **Eventual Consistency**: Guarantees that all replicas will eventually converge to the same state by fully syncing a replica on startup and by being notified by the leader for every update.  (should communicate through a message queue)
- **Periodic persistance**: in case of restarts leader will restore its backup file, replicas will ask for a full sync from leader.


### Run

For production deployments, ensure that each node (whether leader or replica) is hosted on a separate machine or container/pod within the cluster to enhance reliability.

### Architecture:

Leader keeps a map with each word count in memory and periodically writes it to filesystem so that it can restore in case of a failure.
When a write arrives it sends them to all replicas /update route async by opening a goroutine for each replica,
but it waits for at least one replica to respond before it continues.

routes:
- POST /post route handler for feeding it text
- GET /sync route handler for replicas to sync from.

Replica, like the leader, keeps a map in memory and receives updates from leader.
On startup it ask the leader for a full sync.

routes:
- GET /wordcount?word=example route to GET counts
- POST "/update" for leader to send word count updates.
  body example: {"hello": 5, "world": 1}

Both have a /health route

### Local Replica

By default in memdb nodes communicates through REST APIs (ideally should be message queue like redis),
but it can also run with local replica which should run alongside(same machine) leader.

local replica reads the backup file of the leader when is notified by leader to update its in-memory copy.
The eventual consistency can be controlled by increasing the frequency(current: 1 second) on which leader backup his in memory db 

To start the database, execute:

```sh
make start-servers
```

or

```sh
make start-servers-local-replicas
```

To stop the database, execute:
```sh
make stop-servers
```
or

```sh
make stop-servers-local-replicas
```

To run performance test agains them run in another terminal:
```sh
make perf

./bin/perf http://localhost:8080 http://localhost:8081 http://localhost:8083 http://localhost:8082 
```


### Test

To run the tests, use:

```sh
make test
```

### Performance tests

To run performance tests, use:

```sh
make run-perf
```

```sh
make run-perf-local-replicas
```

or 

```sh
make start-server
```

### Docker

Check examples/ dir for docker files.

### Things to improve

- leader to communicate with replicas using a message queue system instead of REST API
- Use gRPC instead of HTTP because the serialization is the most intensive task of memdb.
- add https support
- add basic auth
- add authorization
- use env variables for configuring leader and replica apps
- write logs to a file in rootDir
- add volume to leader dockerfile and docker-compose files.
- write helm charts for k8s deployment
- add CI/CD github workflows
- additional make targets for releasing and deploying
- let the replicas handle multiple words requests
- extend functionalities to meet a database needs :D.

### Performance comparison between replica and local replicas:

with replicas:
```
=== Leader Performance ===
Total Requests to Leader: 1000
Successful Requests: 1000
Failed Requests: 0
Average Request Time: 252.672028ms
Fastest Request Time: 59.391601ms
Slowest Request Time: 479.965803ms

=== Replica Performance ===
Total Requests to Replicas: 1000
Successful Requests: 999
Failed Requests: 0
Average Request Time: 47.766365ms
Fastest Request Time: 9.988ms
Slowest Request Time: 89.9609ms

Total Test Duration: 579.252803ms
All replicas are consistent with the leader.

Test completed
```

with local replicas:
```
=== Leader Performance ===
Total Requests to Leader: 1000
Successful Requests: 1000
Failed Requests: 0
Average Request Time: 323.198353ms
Fastest Request Time: 13.662201ms
Slowest Request Time: 815.86525ms

=== Replica Performance ===
Total Requests to Replicas: 1000
Successful Requests: 999
Failed Requests: 0
Average Request Time: 40.348159ms
Fastest Request Time: 18.672201ms
Slowest Request Time: 105.463407ms

Total Test Duration: 936.313157ms
Inconsistency found for word load: leader count = 2150, replica http://localhost:8082 count = 2105
Inconsistency found for word go: leader count = 8603, replica http://localhost:8081 count = 8398
Inconsistency found for word load: leader count = 2150, replica http://localhost:8081 count = 2105
Inconsistency found for word requests: leader count = 2150, replica http://localhost:8082 count = 2099
Inconsistency found for word go: leader count = 8603, replica http://localhost:8082 count = 8398
Inconsistency found for word is: leader count = 2150, replica http://localhost:8081 count = 2094
Inconsistency found for word test: leader count = 4300, replica http://localhost:8081 count = 4214
Inconsistency found for word test: leader count = 4300, replica http://localhost:8082 count = 4214
Inconsistency found for word in: leader count = 2150, replica http://localhost:8082 count = 2094
Inconsistency found for word concurrency: leader count = 4300, replica http://localhost:8081 count = 4193
Inconsistency found for word requests: leader count = 2150, replica http://localhost:8081 count = 2099
Inconsistency found for word performance: leader count = 2150, replica http://localhost:8081 count = 2109
Inconsistency found for word distributed: leader count = 4303, replica http://localhost:8082 count = 4201
Inconsistency found for word high: leader count = 2150, replica http://localhost:8081 count = 2105
Inconsistency found for word of: leader count = 2153, replica http://localhost:8082 count = 2111
Inconsistency found for word parallel: leader count = 2150, replica http://localhost:8082 count = 2099
Inconsistency found for word model: leader count = 2150, replica http://localhost:8082 count = 2099
Inconsistency found for word model: leader count = 2150, replica http://localhost:8081 count = 2099
Inconsistency found for word great: leader count = 2150, replica http://localhost:8082 count = 2094
Inconsistency found for word world: leader count = 4306, replica http://localhost:8081 count = 4208
Inconsistency found for word high: leader count = 2150, replica http://localhost:8082 count = 2105
Inconsistency found for word parallel: leader count = 2150, replica http://localhost:8081 count = 2099
Inconsistency found for word hello: leader count = 4303, replica http://localhost:8081 count = 4198
Inconsistency found for word hello: leader count = 4303, replica http://localhost:8082 count = 4198
Inconsistency found for word is: leader count = 2150, replica http://localhost:8082 count = 2094
Inconsistency found for word of: leader count = 2153, replica http://localhost:8081 count = 2111
Inconsistency found for word great: leader count = 2150, replica http://localhost:8081 count = 2094
Inconsistency found for word world: leader count = 4306, replica http://localhost:8082 count = 4208
Inconsistency found for word performance: leader count = 2150, replica http://localhost:8082 count = 2109
Inconsistency found for word systems: leader count = 4303, replica http://localhost:8081 count = 4201
Inconsistency found for word systems: leader count = 4303, replica http://localhost:8082 count = 4201
Inconsistency found for word concurrency: leader count = 4300, replica http://localhost:8082 count = 4193
Inconsistency found for word in: leader count = 2150, replica http://localhost:8081 count = 2094
Inconsistency found for word distributed: leader count = 4303, replica http://localhost:8081 count = 4201
Inconsistency found for word go: leader count = 8603, replica http://localhost:8083 count = 8398
Inconsistency found for word systems: leader count = 4303, replica http://localhost:8083 count = 4201
Inconsistency found for word parallel: leader count = 2150, replica http://localhost:8083 count = 2099
Inconsistency found for word requests: leader count = 2150, replica http://localhost:8083 count = 2099
Inconsistency found for word in: leader count = 2150, replica http://localhost:8083 count = 2094
Inconsistency found for word high: leader count = 2150, replica http://localhost:8083 count = 2105
Inconsistency found for word of: leader count = 2153, replica http://localhost:8083 count = 2111
Inconsistency found for word performance: leader count = 2150, replica http://localhost:8083 count = 2109
Inconsistency found for word world: leader count = 4306, replica http://localhost:8083 count = 4208
Inconsistency found for word model: leader count = 2150, replica http://localhost:8083 count = 2099
Inconsistency found for word load: leader count = 2150, replica http://localhost:8083 count = 2105
Inconsistency found for word is: leader count = 2150, replica http://localhost:8083 count = 2094
Inconsistency found for word hello: leader count = 4303, replica http://localhost:8083 count = 4198
Inconsistency found for word test: leader count = 4300, replica http://localhost:8083 count = 4214
Inconsistency found for word distributed: leader count = 4303, replica http://localhost:8083 count = 4201
Inconsistency found for word concurrency: leader count = 4300, replica http://localhost:8083 count = 4193
Inconsistency found for word great: leader count = 2150, replica http://localhost:8083 count = 2094
Found 51 inconsistencies between leader and replicas.

Test completed
```

