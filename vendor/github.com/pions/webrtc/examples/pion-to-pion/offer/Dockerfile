FROM golang:1.11

COPY . /go/src/pion-to-pion/offer
WORKDIR /go/src/pion-to-pion

RUN apt-get update && apt-get install -y \
  libssl-dev

RUN go get -u github.com/pions/webrtc

RUN go install -v ./...

CMD ["offer"]
