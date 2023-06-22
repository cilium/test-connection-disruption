FROM golang:1.20 as gobuilder
WORKDIR /src/test-connection-disruption
COPY . .
RUN make

FROM busybox
COPY --from=gobuilder /src/test-connection-disruption/client /usr/bin/tcd-client
COPY --from=gobuilder /src/test-connection-disruption/server /usr/bin/tcd-server
