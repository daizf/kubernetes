package secret

import (
	"fmt"
	"sync"

	v1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
)

type keyObject struct {
	namespace string
	name      string
}

var ecism = &eciSecretManager{
	store: make(map[keyObject]*v1.Secret),
}

// eciSecretManager implements ConfigMap Manager interface with
// simple operations to ECI.
type eciSecretManager struct {
	store   map[keyObject]*v1.Secret
	locker  sync.RWMutex
	manager Manager
}

// NewEciSecretManager create a configmap manager for ECI.
func NewEciSecretManager() Manager {
	return ecism
}

// SetManager for configmap manager
func (m *eciSecretManager) SetManager(manager Manager) {
	m.locker.Lock()
	defer m.locker.Unlock()
	m.manager = manager
}

func (m *eciSecretManager) getFromLocal(namespace, name string) (*v1.Secret, error) {
	key := keyObject{
		namespace: namespace,
		name:      name,
	}
	secret, ok := m.store[key]
	if !ok {
		return nil, api_errors.NewNotFound(v1.Resource("secret"), fmt.Sprintf("%v/%v", namespace, name))
	}

	return secret, nil
}

func (m *eciSecretManager) UpdateSecret(secrets []v1.Secret) {
	m.locker.Lock()
	defer m.locker.Unlock()

	for i := range secrets {
		secret := secrets[i]
		key := keyObject{secret.Namespace, secret.Name}
		m.store[key] = &secret
	}
}

func (m *eciSecretManager) DeleteSecrets(secrets []v1.Secret) {
	m.locker.Lock()
	defer m.locker.Unlock()

	for i := range secrets {
		secret := secrets[i]
		key := keyObject{secret.Namespace, secret.Name}
		delete(m.store, key)
	}
}

func (m *eciSecretManager) GetSecret(namespace, name string) (*v1.Secret, error) {
	m.locker.RLock()
	defer m.locker.RUnlock()

	configMap, err := m.getFromLocal(namespace, name)
	if api_errors.IsNotFound(err) && m.manager != nil {
		return m.manager.GetSecret(namespace, name)
	}
	return configMap, err
}

func (m *eciSecretManager) RegisterPod(pod *v1.Pod) {
	m.locker.Lock()
	defer m.locker.Unlock()

	if m.manager != nil {
		m.manager.RegisterPod(pod)
	}
}

func (m *eciSecretManager) UnregisterPod(pod *v1.Pod) {
	m.locker.Lock()
	defer m.locker.Unlock()

	if m.manager != nil {
		m.manager.UnregisterPod(pod)
	}
}

func (m *eciSecretManager) Clean() {
	m.locker.Lock()
	defer m.locker.Unlock()
	m.store = make(map[keyObject]*v1.Secret)
}

func SetManager(manager Manager) {
	ecism.SetManager(manager)
}

func EciAddSecrets(secrets []v1.Secret) {
	ecism.UpdateSecret(secrets)
}

func EciDeleteSecrets(secrets []v1.Secret) {
	ecism.DeleteSecrets(secrets)
}

func EciGetSecret(namespace, name string) (*v1.Secret, error) {
	return ecism.GetSecret(namespace, name)
}

func EciCleanSecrets() {
	ecism.Clean()
}
