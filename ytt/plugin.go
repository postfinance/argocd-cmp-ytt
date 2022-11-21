// Package main is the ArgoCD extension of argoCD as a ConfigManagementPlugin that permits templating manifests with ytt
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bitfield/script"
)

//nolint:gochecknoglobals // simpler to debug using these globals
var (
	indent       = "    "
	debugContent = make([]string, 0, 300)
)

func toDebug(str string) {
	for _, s := range strings.Split(str, "\n") {
		debugContent = append(debugContent, fmt.Sprintf("%s%s", indent, s))
	}
}

func renderDebugAndExit() {
	fmt.Print(`
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-cmp-ytt-error
  namespace: debug
stringData:
  debug.txt: |
`)

	for _, str := range debugContent {
		fmt.Println(str)
	}

	os.Exit(0)
}

func renderHelmChart() {
	appNs, _ := os.LookupEnv("ARGOCD_APP_NAMESPACE")
	appName, _ := os.LookupEnv("ARGOCD_APP_NAME")

	if appNs == "" {
		toDebug("#! the namespace of the application isn't defined. not possible to render the Helm Chart without that.")
		renderDebugAndExit()
	}

	// Helm Chart detected. we only apply ytt on values.yaml and then use helm to template
	yttPipe := script.Exec("ytt --data-values-env=ARGOCD_ENV --file values.yaml --output-files .")
	yttPipe.Wait()
	err := yttPipe.Error()

	if err != nil {
		toDebug(fmt.Sprintf("#! cmp-ytt: an error occurred while rendering the ytt template: %v", err))
		renderDebugAndExit()
	}

	stdout, err := script.Exec("helm dependency build").String()
	if err != nil {
		toDebug(fmt.Sprintf("#! cmp-ytt: error while pulling helm depencies: %v", err))
		toDebug(stdout)
		renderDebugAndExit()
	}

	stdout, err = script.Exec(fmt.Sprintf("helm template %s --namespace %s .", appName, appNs)).String()
	if err != nil {
		toDebug("#! cmp-ytt: error while templating/rendering the Helm Chart")
		toDebug(stdout)
		renderDebugAndExit()
	}

	fmt.Print(stdout)
}

func renderKustomization() {
	// Kustomization detected - we apply ytt on all files and then use kustomize build
	stdout, err := script.Exec("ytt --data-values-env=ARGOCD_ENV --file . --output-files .").String()
	if err != nil {
		toDebug(fmt.Sprintf("#cmp-ytt: an error occurred while rendering the ytt template: %v", err))
		toDebug(stdout)
		renderDebugAndExit()
	}

	stdout, err = script.Exec("kustomize build .").String()
	if err != nil {
		toDebug("#! error while running kustomize build:")
		toDebug(stdout)
		renderDebugAndExit()
	}

	fmt.Print(stdout)
}

func main() {
	files, err := os.ReadDir(".")
	if err != nil {
		toDebug("# Couldn't read files!")
		toDebug(err.Error())
		renderDebugAndExit()
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		if f.Name() == "Chart.yaml" {
			renderHelmChart()
			return
		} else if isKustomization(f.Name()) {
			renderKustomization()
			return
		}
	}

	// no Helm Chart nor Kustomization detected - rendering plain manifests
	stdout, err := script.Exec("ytt --data-values-env=ARGOCD_ENV --file .").String()
	if err != nil {
		toDebug(fmt.Sprintf("#! cmp-ytt: an error occurred while rendering the ytt template: %v", err))
		toDebug(stdout)
	}

	fmt.Print(stdout)
}

func isKustomization(path string) bool {
	// source: https://github.com/argoproj/argo-cd/blob/v2.4.14/util/kustomize/kustomize.go#L208
	var kustomizationNames = []string{"kustomization.yaml", "kustomization.yml", "Kustomization"}

	for _, kustomization := range kustomizationNames {
		if path == kustomization {
			return true
		}
	}

	return false
}
