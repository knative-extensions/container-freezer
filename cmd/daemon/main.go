package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/kelseyhightower/envconfig"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"knative.dev/container-freezer/pkg/daemon"
	"knative.dev/container-freezer/pkg/freeze/containerd"
	pkglogging "knative.dev/pkg/logging"
)

const runtimeTypeContainerd string = "containerd"

type config struct {
	RuntimeType string `split_words:"true" required:"true"`

	// Logging configuration
	FreezerLoggingConfig string `split_words:"true"`
	FreezerLoggingLevel  string `split_words:"true"`
}

func main() {
	// Parse the environment.
	var env config
	if err := envconfig.Process("", &env); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logger, _ := pkglogging.NewLogger(env.FreezerLoggingConfig, env.FreezerLoggingLevel)
	runtimeType := env.RuntimeType

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
		// TODO support crio
	default:
		log.Fatal("unrecognised runtimeType", runtimeType)
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
