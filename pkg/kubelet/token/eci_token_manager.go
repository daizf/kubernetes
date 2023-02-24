package token

// Package token implements a manager of serviceaccount tokens for pods running on the ECI.
import (
	"context"
	"errors"
	"fmt"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"k8s.io/utils/clock"
)

type ServiceAccountTokenCreator interface {
	CreateServiceAccountToken(ctx context.Context, namespace, name string, tr *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error)
}

// EciTokenManager manages service account tokens and kubeclient for pods.
type EciTokenManager struct {
	// TokenManager
	*Manager

	creator ServiceAccountTokenCreator
}

var ecitm = newEciTokenManager()

// NewEciTokenManager creates a token manager for ECI.
func NewEciTokenManager() *EciTokenManager {
	return ecitm
}

// NewEciTokenManager returns a new token manager.
func newEciTokenManager() *EciTokenManager {
	tokenManager := &Manager{
		cache: make(map[string]*authenticationv1.TokenRequest),
		clock: clock.RealClock{},
	}
	eciTokenManager := &EciTokenManager{
		Manager: tokenManager,
	}
	go wait.Forever(eciTokenManager.cleanup, gcPeriod)
	return eciTokenManager
}

// SetServiceAccountTokenCreator for token manager
func SetServiceAccountTokenCreator(creator ServiceAccountTokenCreator) {
	ecitm.cacheMutex.Lock()
	defer ecitm.cacheMutex.Unlock()
	ecitm.creator = creator
}

func getToken(name, namespace string, tr *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
	creator := ecitm.creator
	if creator == nil {
		return nil, errors.New("cannot use TokenManager when kubelet is in standalone mode")
	}
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()
	tokenRequest, err := creator.CreateServiceAccountToken(ctx, namespace, name, tr)
	return tokenRequest, err
}

func (m *EciTokenManager) GetServiceAccountToken(namespace, name string, tr *authenticationv1.TokenRequest) (*authenticationv1.TokenRequest, error) {
	key := keyFunc(name, namespace, tr)

	ctr, ok := m.get(key)

	if ok && !m.requiresRefresh(ctr) {
		return ctr, nil
	}

	tr, err := getToken(name, namespace, tr)
	if err != nil {
		switch {
		case !ok:
			return nil, fmt.Errorf("failed to fetch token: %v", err)
		case m.expired(ctr):
			return nil, fmt.Errorf("token %s expired and refresh failed: %v", key, err)
		default:
			klog.Errorf("couldn't update token %s: %v", key, err)
			return ctr, nil
		}
	}

	m.set(key, tr)
	return tr, nil
}
