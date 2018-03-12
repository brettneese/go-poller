FROM centurylink/ca-certs

COPY go-poller /
ENTRYPOINT ["/go-poller"]
