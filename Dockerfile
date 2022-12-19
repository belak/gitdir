# Stage 1: Build the application
FROM golang:1.19-bullseye as builder

RUN mkdir /build && mkdir /usr/local/src/gitdir
WORKDIR /usr/local/src/gitdir

ADD ./go.mod ./go.sum ./
RUN go mod download
ADD . ./

RUN go build -v -o /build/gitdir ./cmd/gitdir

# State 2: Copy files and configure what we need
FROM debian:bullseye-slim as runner

ENV GITDIR_BASE_DIR=/var/lib/gitdir \
  GITDIR_BIND_ADDR=0.0.0.0:2222

VOLUME /var/lib/gitdir

# Install git so git-upload-pack and git-receive-pack are available.
RUN apt-get update && apt-get install -y git \
  && rm -rf /var/lib/apt/lists/*

COPY --from=builder /build/gitdir /usr/bin/gitdir

EXPOSE 2222
CMD ["gitdir"]
