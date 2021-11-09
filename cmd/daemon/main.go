package main

import (
	"context"
	"log"
	"net/http"
	"os"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"knative.dev/container-freezer/pkg/daemon"
	"knative.dev/container-freezer/pkg/freeze/containerd"
	"knative.dev/container-freezer/pkg/freeze/docker"
	pkglogging "knative.dev/pkg/logging"
)

const (
	runtimeTypeContainerd string = "containerd"
	runtimeTypeDocker     string = "docker"
)

func main() {
	logger, _ := pkglogging.NewLogger("", "") // TODO read from envvar
	runtimeType := os.Getenv("RUNTIME_TYPE")

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	var freezeThaw daemon.FreezeThawer
	switch runtimeType {
	case runtimeTypeContainerd:
		logger.Info("creating new containerd freezeThawer")
		ctrd, err := containerd.NewCRI()
		if err != nil {
			log.Fatalf("unable to create containerd cri: %v", err)
		}
		freezeThaw = containerd.New(ctrd)
	case runtimeTypeDocker:
		logger.Info("creating new docker freezeThawer")
		dkr, err := docker.NewCRI()
		if err != nil {
			log.Fatalf("unable to create docker cri: %v", err)
		}
		freezeThaw = docker.New(dkr)
		// TODO support crio
	default:
		log.Fatal("unrecognised runtimeType", runtimeType)
	}
	if err != nil {
		log.Fatal(err)
	}

	http.ListenAndServe(":8080", &daemon.Handler{
		Freezer: freezeThaw,
		Thawer:  freezeThaw,
		Logger:  logger,
		Validator: daemon.TokenValidatorFunc(func(ctx context.Context, token string) (*authv1.TokenReview, error) {
			return clientset.AuthenticationV1().TokenReviews().Create(ctx, &authv1.TokenReview{
				Spec: authv1.TokenReviewSpec{
					Token: token,
					Audiences: []string{
						// The projected token only gives the right to pause/resume
						"concurrency-state-hook",
					},
				},
			}, metav1.CreateOptions{})
		}),
	})
}
