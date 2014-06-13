# Forerunner

Forerunner is a daemon to announce and check Docker services, to be run in a
cluster with something like Fleet.

## Usage

```
Usage of forerunner:
      --config-prefix="/forerunner/": etcd prefix to pull configuration from
      --dns="": DNS host to use for container
      --docker-endpoint="unix:///var/run/docker.sock": docker socket to use
      --etcd-hosts="http://127.0.0.1:4001": comma-separated list of etcd hosts to connect to
      --group="": this service's group
      --id="": this service's ID
      --image="": docker image to run
      --log-level="info": level to log at (debug, info, warning, error, fatal)
      --public-host="127.0.0.1": public IP for service routing
      --public-port=0: public port for service routing
      --register-vulcan=false: register a vulcan endpoint for this service
      --registry="": docker registry to contact
      --shutdown-timeout=5s: how long to wait after interrupt before forcibly stopping
      --tcp-health-host="127.0.0.1": container host
      --tcp-health-port=0: container port to check over TCP
```
