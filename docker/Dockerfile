FROM docker.io/library/alpine:3.24 as runtime

ENTRYPOINT ["kubernetes-zfs-provisioner"]

RUN \
    apk add --no-cache curl bash && \
    adduser -S zfs -G root

COPY kubernetes-zfs-provisioner /usr/bin/

USER zfs:root
