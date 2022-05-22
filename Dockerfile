
FROM golang:alpine as builder
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN apk add --no-cache git && go build -o sepio-bot . && apk del git

FROM alpine
WORKDIR /app
COPY --from=builder /app/sepio-bot .
CMD [ "./sepio-bot" ]