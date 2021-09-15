# migrate-svc-test

This repo contains dummy echo TCP and UDP servers and clients
that are used to test whether connections are not interrupted 
during cases such as Cilium upgrades and LB service endpoint updates.

A container image containing both can be fetched from
[here](https://hub.docker.com/r/cilium/migrate-svc-test).

## Build

```
$ VERSION=v0.0.2 ./build.sh
```
