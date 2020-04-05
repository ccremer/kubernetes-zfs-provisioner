FROM docker.io/library/alpine:3.11 as runtime

ENTRYPOINT ["kubernetes-zfs-provisioner"]

RUN \
    apk add --no-cache curl bash zfs

COPY kubernetes-zfs-provisioner /usr/bin/
