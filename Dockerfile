# BUILDER 
FROM golang:1.16-alpine3.12 AS builder
ENV GOOS=linux
WORKDIR /app
COPY . /app
RUN go build 

# RELEASE
FROM alpine:3.12

RUN  apk add --no-cache --update ca-certificates

COPY --from=builder /app/kube-sqs-autoscaler ./

CMD ["/kube-sqs-autoscaler"]
