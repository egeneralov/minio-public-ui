FROM golang:1.18
WORKDIR /go/src/github.com/egeneralov/minio-public-ui/
ARG GOPROXY
ADD . .
RUN go build -v -ldflags "-v -linkmode auto -extldflags \"-static\"" -o /go/bin/minio-public-ui github.com/egeneralov/minio-public-ui/cmd/minio-public-ui

FROM debian:bullseye
RUN apt-get update -q && apt-get install -yq ca-certificates --no-install-recommends
ENV PATH='/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin'
CMD /go/bin/minio-public-ui
COPY --from=0 /go/bin /go/bin
