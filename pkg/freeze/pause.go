package freeze

import (
	"context"
	"fmt"

	"knative.dev/container-freezer/pkg/freeze/containerd"
	"knative.dev/container-freezer/pkg/freeze/crio"
)

const (
	runtimeTypeContainerd = "containerd"
	runtimeTypeCrio       = "crio"
)

type CRI interface {
	List(ctx context.Context, podUID string) ([]string, error)
	Pause(ctx context.Context, container string) error
	Resume(ctx context.Context, container string) error
}

type CRIImpl struct {
	cri CRI
}

func NewCRIProvider(runtimeType string) (*CRIImpl, error) {
	criImpl := &CRIImpl{}

	switch runtimeType {
	case runtimeTypeContainerd:
		containerdImpl, err := containerd.NewContainerdProvider()
		if err != nil {
			return nil, err
		}
		criImpl.cri = containerdImpl
		return criImpl, err
	case runtimeTypeCrio:
		crioImpl, err := crio.NewCrioProvider()
		if err != nil {
			return nil, err
		}
		criImpl.cri = crioImpl
		return criImpl, err
	default:
		return nil, fmt.Errorf("unrecognised runtimeType:%s", runtimeType)
	}
}

func (c *CRIImpl) Freeze(ctx context.Context, podName string) error {
	containerIDs, err := c.cri.List(ctx, podName)
	if err != nil {
		return err
	}

	for _, ctr := range containerIDs {
		if err := c.cri.Pause(ctx, ctr); err != nil {
			return fmt.Errorf("%s not paused: %v", ctr, err)
		}
	}
	return nil
}

func (c *CRIImpl) Thaw(ctx context.Context, podName string) error {
	containerIDs, err := c.cri.List(ctx, podName)
	if err != nil {
		return err
	}

	for _, ctr := range containerIDs {
		if err := c.cri.Resume(ctx, ctr); err != nil {
			return fmt.Errorf("%s not resumed: %v", ctr, err)
		}
	}
	return nil
}
