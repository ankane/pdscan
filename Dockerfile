FROM scratch
ENTRYPOINT ["/pdscan"]
COPY pdscan /
COPY licenses /licenses
