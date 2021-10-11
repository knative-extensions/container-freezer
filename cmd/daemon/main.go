package main

import (
	"log"
	"net/http"

	"knative.dev/container-freezer/pkg/daemon"
	pkglogging "knative.dev/pkg/logging"
)

const runtimeTypeContainerd string = "containerd"

func main() {
	logger, _ := pkglogging.NewLogger("", "")
	logger.Info("handling requests to freeze daemon")
	runtimeType := runtimeTypeContainerd

	switch runtimeType {
	case runtimeTypeContainerd:
		logger.Info("creating new containerd freezer")
		// TODO suport docker, crio
	default:
		log.Fatal("unrecognised runtimeType", runtimeType)
	}

	http.ListenAndServe(":9696", &daemon.Handler{
		Logger: logger,
	})
}
