# migrate-svc-test

This repo contains a dummy TCP server and client which are used to test Cilium
service migrations from legacy to v2 for the v1.5 release.

A container image containing both can be fetched from
[here](https://hub.docker.com/r/cilium/migrate-svc-test).

## Build

```
$ VERSION=v0.0.1 ./build.sh
```
