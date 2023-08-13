# kubernetes-zfs-provisioner

![Version: 2.2.1](https://img.shields.io/badge/Version-2.2.1-informational?style=flat-square)

Dynamic ZFS persistent volume provisioner for Kubernetes

## Installation

```bash
helm repo add zfs-provisioner https://ccremer.github.io/kubernetes-zfs-provisioner
helm install kubernetes-zfs-provisioner zfs-provisioner/kubernetes-zfs-provisioner
```

<!---
The README.md file is automatically generated with helm-docs!

Edit the README.gotmpl.md template instead.
-->

<!---
The values below are generated with helm-docs!

Document your changes in values.yaml and let `make docs:helm` generate this section.
-->
## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| env | object | `{}` | A dict with KEY: VALUE pairs |
| fullnameOverride | string | `""` |  |
| hostAliases | object | `{}` | A dict with `{ip, hostnames array}` to configure custom entries in /etc/hosts. See [values.yaml](./values.yaml) for an example. |
| image.pullPolicy | string | `"Always"` |  |
| image.registry | string | `"ghcr.io"` | Container image registry |
| image.repository | string | `"ccremer/zfs-provisioner"` | Location of the container image |
| image.tag | string | `"v1"` | Container image tag |
| imagePullSecrets | list | `[]` |  |
| nameOverride | string | `""` |  |
| nodeSelector | object | `{}` | Reminder: This has no effect on any PVs, but maybe you want the provisioner pod running on certain nodes. |
| podSecurityContext | object | `{}` | If you encounter **issues with SSH, set `podSecurityContext.fsGroup=100`**, as the SSH files might not be readable to the container user `zfs` with uid 100. |
| provisioner.instance | string | `"pv.kubernetes.io/zfs"` | Provisoner instance name if multiple are running (multiple instances are not required for managing multiple ZFS hosts) |
| rbac.create | bool | `false` | **Required for first time deployments** Grant the service account the necessary permissions, |
| replicaCount | int | `1` | Usually `1` is fine |
| resources.limits.memory | string | `"40Mi"` |  |
| resources.requests.cpu | string | `"50m"` |  |
| resources.requests.memory | string | `"20Mi"` |  |
| securityContext | object | `{}` |  |
| serviceAccount.annotations | object | `{}` | Annotations to add to the service account |
| serviceAccount.create | bool | `true` | Specifies whether a service account should be created |
| serviceAccount.name | string | `""` | The name of the service account to use. If not set and create is true, a name is generated using the fullname template |
| ssh.config | string | `""` | **Required.** ssh_config(5)-compatible file content to configure SSH options when connecting |
| ssh.externalSecretName | string | `""` | If SSH secrets are managed externally, specify the name |
| ssh.identities | object | `{}` | **Required.** Provide a private key for each SSH identity. See [values.yaml](./values.yaml) for an example |
| ssh.knownHosts | list | `[]` | **Required.** List of {host, pubKey} dicts where the public key of each host is configured |
| ssh.mountPath | string | `"/home/zfs/.ssh"` | The path where the SSH config and identities are mounted |
| storageClass.classes | list | `[]` | Storage classes to create. See [values.yaml](values.yaml) for an example. |
| storageClass.create | bool | `false` | Whether to create storage classes for this provisioner. |
| tolerations | list | `[]` |  |

## Upgrading from 0.x to 1.x charts

There are some breaking changes from 0.x to 1.x versions.

* The `storageclass.classes` array is now empty.
  Where it previously contained an example, the example is removed as a default value.
  The example is still in `values.yaml` in form of YAML comments.
* The `image.registry` has changed from `docker.io` to `quay.io` due to Docker Hub's pull limit.
* Bumped `image.tag` to `v1.0.0`

## Upgrading from 1.x to 2.x charts

Due to the migration of the [chart from ccremer/charts](https://github.com/ccremer/charts/tree/master/charts/kubernetes-zfs-provisioner) to this repo, a breaking change was made for the chart.
Only chart archives from version 2.x can be downloaded from the https://ccremer.github.io/kubernetes-zfs-provisioner index.
No 1.x or 0.x chart releases will be migrated from the `ccremer/charts` Helm repo.

* The `image.registry` has changed from `quay.io` to `ghcr.io`.
* The `image.tag` has changed from to `v1.1.0` to `v1`.
* The `image.pullPolicy` has changed from `IfNotPresent` to `Always`.
