FROM alpine:3.13

ARG TARGETARCH
RUN apk --no-cache add ca-certificates
RUN mkdir -p /var/runtime
COPY bootstrap.${TARGETARCH} /var/runtime/bootstrap
WORKDIR /var/runtime
CMD ["/var/runtime/bootstrap"]
