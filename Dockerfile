FROM golang:1.10.3 AS build
WORKDIR /go/src/github.com/scriptonist/hashnodebot

RUN go get github.com/golang/dep/cmd/dep
COPY Gopkg.toml Gopkg.lock ./
RUN dep ensure -v -vendor-only
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o /go/bin/hashnodebot -ldflags="-w -s" -v github.com/scriptonist/hashnodebot


FROM alpine:3.7 AS final
RUN apk --no-cache add ca-certificates
COPY --from=build /go/bin/hashnodebot /bin/hashnodebot
