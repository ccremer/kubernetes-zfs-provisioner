
<!---
The README.md file is automatically generated with helm-docs!

Edit the README.gotmpl.md template instead.
-->

<!---
The values below are generated with helm-docs!

Document your changes in values.yaml and let `make docs:helm` generate this section.
-->
{{ template "chart.valuesSection" . }}

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
