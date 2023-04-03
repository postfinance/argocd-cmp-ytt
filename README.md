# ArgoCD `ytt` Config Management Plugin (CMP)

This repository contains an [ArgoCD configuration management
plugin](https://argo-cd.readthedocs.io/en/stable/user-guide/config-management-plugins/)
(CMP) for `ytt` which permits using [*YAML Templating Tookit*](https://carvel.dev/ytt/) to template YAML/manifests before they are applied by ArgoCD.

More information on CMP can be read [in the ArgoCD documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/config-management-plugins/).

What's special with this `argocd-cmp-ytt` configuration management plugin is that it permits chaining `ytt` and `Helm`/`Kustomize`, which makes it possible to use the advanced templating capabilities of `ytt` in e.g. the `values.yaml` or in a `kustomization.yaml` (or any other file for that matter), before then running `helm template` or `kustomize build` in the repo.

## How to use it?

1. add the sidecar container to your repo-server deployment (this can be achieved with the file in [`./packaging/k8s/`](./packaging/k8s/), which you can apply to your repo-server deployment as a strategic merge patch if you're using kustomize)
2. add a `.ytt` file (can be empty) at the root repo configured for your application
3. add a `plugin:` section to your application spec, to add some environment variables

an example `Application` would look like follows:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: kube-contour
  namespace: kube-argocd-master
spec:
  destination:
    namespace: kube-contour
    server: https://yourcluster.company.local
  project: kube-contour-appset
  source:
    repoURL: git@gitlab.pnet.ch:kubernetes/workloads/kube-contour.git
    path: .
    plugin:
      env:
      - name: ingress_name
        value: ingress.yourcluster.company.local
      - name: HELM_RELEASE_NAME # optional, can be used to specify the Helm Release name
        value: contour
    targetRevision: HEAD
```

Thanks to the `.ytt` file in your repository, ArgoCD repository server will
know that the repository is to be rendered using the `argocd-cmp-ytt` plugin,
and it will then use the sidecar container to render the end manifests.

As specified above, this plugin not only makes it possible to template using
ytt, it also permits rendering manifests with `Helm` or `kustomize` afterwards.

### templating + `Helm` or `kustomize`

when `argocd-cmp-ytt` is called, it lists all files/folders in the repository
and checks whether:

- there is a `kustomization.yaml` file: in that case, the plugin calls
 `ytt --data-values-env=ARGOCD_ENV` in-place on all files before running `kustomize build .`
- there is a `Chart.yaml` file: in that case, the plugin calls `ytt` in-place
  only on the `values.yaml` file. it then runs `helm dependency build` and
  finally runs `helm template ...`
- there isn't any of the above, it then simply calls
  `ytt --data-values-env=ARGOCD_ENV --file .`

## what can be achived with `ytt` ?

`ytt` [playground](https://carvel.dev/ytt/#playground) and [documentation](https://carvel.dev/ytt/docs/v0.44.0/) are the places where you'll find examples and ideas to template your manifests.

Here is however a simple example showing what can be attained with `ytt` and how it solves a concrete manifest rendering pain point we had:

```yaml
#@ load("@ytt:data", "data")
---
#@yaml/text-templated-strings
ingress_name: "(@= data.values.ingress_name @)" # some simple templating taking place
contour:
  enabled: true
  podAnnotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8000"
  #@yaml/text-templated-strings
  args:
    - serve
    - --incluster
    - --xds-address=0.0.0.0
    - --xds-port=8001
    - --ingress-status-address={{ getHostByName "(@= data.values.ingress_name @)" }} # we can make a DNS lookup here!
```
