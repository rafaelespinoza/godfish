FROM golang:alpine
WORKDIR /src
RUN apk update && apk --no-cache add gcc g++ git make
COPY go.mod /src
RUN go mod download && go mod verify
COPY . /src
