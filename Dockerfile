# syntax=docker/dockerfile:1
FROM scratch
ENTRYPOINT ["/pdscan"]
ARG VERSION
ADD --unpack=true https://github.com/ankane/pdscan/releases/download/v$VERSION/pdscan-$VERSION-x86_64-linux.tar.gz /
