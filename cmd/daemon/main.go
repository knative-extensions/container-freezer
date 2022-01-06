package main

import (
	"context"
	"log"
	"net/http"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"knative.dev/container-freezer/pkg/api"
	"knative.dev/container-freezer/pkg/daemon"
	"knative.dev/container-freezer/pkg/freeze/containerd"
	pkglogging "knative.dev/pkg/logging"
)

func main() {
	logger, _ := pkglogging.NewLogger("", "") // TODO read from envvar
	runtimeType := api.RuntimeTypeContainerd  // TODO read from envvar

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	var freezeThaw daemon.FreezeThawer
	switch runtimeType {
	case api.RuntimeTypeContainerd:
		logger.Info("creating new containerd freezeThawer")
		ctrd, err := containerd.NewCRI()
		if err != nil {
			log.Fatalf("unable to create containerd cri: %v", err)
		}
		freezeThaw = containerd.New(ctrd)
		// TODO support docker, crio
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
			return clientSet.AuthenticationV1().TokenReviews().Create(ctx, &authv1.TokenReview{
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
