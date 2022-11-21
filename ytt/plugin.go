package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/bitfield/script"
)

var debugConfigMap string = `
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-cmp-ytt-error
  namespace: debug
stringData:
  debug.txt: |
`
var indent string = "    "
var debugContent []string = make([]string, 0, 300)

func toDebug(str string) {
	for _, s := range strings.Split(str, "\n") {
		debugContent = append(debugContent, fmt.Sprintf("%s%s", indent, s))
	}
}

func renderDebugAndExit() {
	fmt.Println(debugConfigMap)
	for _, str := range debugContent {
		fmt.Println(str)
	}
	os.Exit(0)
}

func renderHelmChart() {
	app_namespace, _ := os.LookupEnv("ARGOCD_APP_NAMESPACE")
	app_name, _ := os.LookupEnv("ARGOCD_APP_NAME")

	if len(app_namespace) == 0 {
		toDebug("#! the namespace of the application isn't defined. not possible to render the Helm Chart without that.")
		renderDebugAndExit()
	}

	// Helm Chart detected. we only apply ytt on values.yaml and then use helm to template
	yttPipe := script.Exec("ytt --data-values-env=ARGOCD_ENV --file values.yaml --output-files .")
	yttPipe.Wait()
	err := yttPipe.Error()
	if err != nil {
		toDebug(fmt.Sprintf("#! cmp-ytt: an error occured while rendering the ytt template: %v", err))
		renderDebugAndExit()
	}

	stdout, err := script.Exec("helm dependency build").String()
	if err != nil {
		toDebug(fmt.Sprintf("#! cmp-ytt: error while pulling helm depencies: %v", err))
		toDebug(stdout)
		renderDebugAndExit()
	}

	stdout, err = script.Exec(fmt.Sprintf("helm template %s --namespace %s .", app_name, app_namespace)).String()
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
		toDebug(fmt.Sprintf("#cmp-ytt: an error occured while rendering the ytt template: %v", err))
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

	debugConfigMap += indent + "# current path\n"
	dir, err := os.Getwd()
	if err != nil {
		toDebug(err.Error())
	}
	toDebug(dir)

	files, err := ioutil.ReadDir(".")
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
		} else if IsKustomization(f.Name()) {
			renderKustomization()
			return
		}
	}

	// no Helm Chart nor Kustomization detected - rendering plain manifests
	stdout, err := script.Exec("ytt --data-values-env=ARGOCD_ENV --file .").String()
	if err != nil {
		toDebug(fmt.Sprintf("#! cmp-ytt: an error occured while rendering the ytt template: %v", err))
		toDebug(stdout)
	}

	fmt.Print(stdout)
}

// source: https://github.com/argoproj/argo-cd/blob/v2.4.14/util/kustomize/kustomize.go#L208
var KustomizationNames = []string{"kustomization.yaml", "kustomization.yml", "Kustomization"}

func IsKustomization(path string) bool {
	for _, kustomization := range KustomizationNames {
		if path == kustomization {
			return true
		}
	}
	return false
}
