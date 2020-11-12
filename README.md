# migrate-svc-test

This repo contains a dummy echo TCP server and client which are used to test
whether connections are not interrupted during Cilium upgrades.

A container image containing both can be fetched from
[here](https://hub.docker.com/r/cilium/migrate-svc-test).

## Build

```
$ VERSION=v0.0.2 ./build.sh
```
