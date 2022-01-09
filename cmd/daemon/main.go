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
	"knative.dev/container-freezer/pkg/freeze/docker"
	pkglogging "knative.dev/pkg/logging"
)

const (
	runtimeTypeContainerd string = "containerd"
	runtimeTypeDocker     string = "docker"
)

type config struct {
	RuntimeType string `envconfig:"RUNTIME_TYPE" required:"true"`
	DockerAPIVersion string `envconfig:"DOCKERAPI_VERSION" required:"false"`

	// Logging configuration
	FreezerLoggingConfig string `envconfig:"FREEZER_LOGGING_CONFIG"`
	FreezerLoggingLevel  string `envconfig:"FREEZER_LOGGING_LEVEL"`
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

	clientSet, err := kubernetes.NewForConfig(config)
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
		dcr, err := docker.NewCRI(env.DockerAPIVersion)
		if err != nil {
			log.Fatalf("unable to create docker cri: %v", err)
		}
		freezeThaw = docker.New(dcr)
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
