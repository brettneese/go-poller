FROM scratch

COPY go-poller /
ENTRYPOINT ["/go-poller"]
