# gRPC prober

This program exposes a simple gRPC service (as defined in the [proto file][p]) that replies when called on the `PingIt` rpc method.
It is meant to be deployed in a mesh to continuously test the health of it on the gRPC layer.
Every replica is a service and a client itself. If instructed at start up, it tries to contact a list of given peers. When any of the peer gets contactet the first time, it tries to also ping back, invoking the `PingIt` rpc call.
It accepts the following environment variables:

 - `NAME`: the name of the current instance to which it is reachable in the mesh
 - `DEBUG`: if set to any value, enables debug logs
 - `FREQUENCY`: how often to ping the peers (follows the usual golang `time.Duration` format)
 - `PEERS`: a semicolon-separated list of peers to be contacted
 - `PORT`: the port at which each peer and itself must expose the gRPC service

## Dev

There is a docker-compose based local dev enviromnent.

```
docker compose build
docker compose up
```

This spins three replicas (`pinger-1`, `pinger-2`, `pinger-3`), configured to ping each other in loop `1->2->3->1`, and a prometheus instance on `localhost:9090`.
To verify that the service behaves as expected, use the following query:

```
rate(grpc_probe_success{dst="pinger-2"}[30s]) or rate(grpc_probe_failure{dst="pinger-2"}[30s])
```

and try shutting down `pinger-2`:
```
docker compose stop pinger-2
```

Notice how the metrics react, then start it again:

```
docker compose start pinger-2
```

[p]: ./proto/service.proto
