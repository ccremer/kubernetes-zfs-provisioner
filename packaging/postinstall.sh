#!/bin/sh

service="zfs-provisioner.service"

/bin/systemctl daemon-reload
/bin/systemctl enable ${service}
/bin/systemctl start ${service}
