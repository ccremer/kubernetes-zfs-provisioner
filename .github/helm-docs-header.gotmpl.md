{{ template "chart.header" . }}
{{ template "chart.deprecationWarning" . }}

{{ template "chart.versionBadge" . }}{{ template "chart.typeBadge" . }}{{ template "chart.appVersionBadge" . }}

{{ template "chart.description" . }}

{{ template "chart.homepageLine" . }}

## Installation

```bash
helm repo add zfs-provisioner https://ccremer.github.io/kubernetes-zfs-provisioner
helm install {{ template "chart.name" . }} zfs-provisioner/{{ template "chart.name" . }}
```
