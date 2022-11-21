# ArgoCD Config Management Plugins (CMP)

This repository contains an [ArgoCD configuration management
plugin](https://argo-cd.readthedocs.io/en/stable/user-guide/config-management-plugins/)
(CMP) for `ytt`, and more plugins might come in the future.  Those can be used
to to extend the manifest rendering capability of ArgoCD, for example to permit
rendering and application `source:` with Google's `kpt`, with `ytt`, with
`jsonnet`, or a combination of templating tools.
