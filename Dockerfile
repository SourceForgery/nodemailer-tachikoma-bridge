FROM ubuntu:22.04  as golang-build
RUN apt-get update && apt-get install -y curl protobuf-compiler
RUN mkdir -p /go/.go/wrapper
WORKDIR /go

ADD gow gow.cmd /go/
ADD .go/wrapper/go-wrapper.properties /go/.go/wrapper/

ADD go.mod go.sum /go/
RUN ./gow install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
RUN ./gow install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
RUN ./gow mod download

ADD . /go/

RUN ./gow generate
RUN ./gow build

FROM ubuntu:22.04
RUN apt-get update && apt-get install -y ca-certificates && apt-get clean && rm -rf /var/lib/apt/lists/*

COPY --from=golang-build /go/tachikoma-bridge /tachikoma-bridge

WORKDIR /
USER www-data:www-data
EXPOSE 8080

ENTRYPOINT ["/tachikoma-bridge"]

