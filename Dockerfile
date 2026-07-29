# BUILDER
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.26 AS builder

ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src/test-connection-disruption
ADD . .
RUN make clean
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} make

# RUNTIME
FROM busybox
COPY --from=builder /src/test-connection-disruption/client /usr/bin/tcd-client
COPY --from=builder /src/test-connection-disruption/server /usr/bin/tcd-server
