ARG ARCH
FROM ${ARCH}/alpine:3.11

ENTRYPOINT ["kubernetes-zfs-provisioner"]

RUN \
    apk add --no-cache curl bash openssh && \
    adduser -S zfs -G root

COPY docker/zfs.sh /usr/bin/zfs
COPY docker/update-permissions.sh /usr/bin/update-permissions
COPY kubernetes-zfs-provisioner /usr/bin/

USER zfs:root
