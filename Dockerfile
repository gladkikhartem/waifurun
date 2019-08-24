FROM ubuntu:18.04 as build

RUN apt-get update && apt-get install -y wget git
ENV PATH="/usr/local/go/bin:${PATH}"
RUN wget https://dl.google.com/go/go1.12.9.linux-amd64.tar.gz && tar -C /usr/local -xzf go1.12.9.linux-amd64.tar.gz
WORKDIR /app
COPY main.go go.mod go.sum  ./
RUN go build -tags netgo -ldflags '-w -extldflags "-static"' -o waifu.run


FROM nothink/waifu2x
RUN mkdir -p /data
COPY --from=build /app/waifu.run .
ENTRYPOINT ["/opt/waifu2x-cpp/waifu.run"]