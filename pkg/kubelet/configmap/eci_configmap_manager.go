package configmap

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

var ecicmm = &eciConfigMapManager{
	store: make(map[keyObject]*v1.ConfigMap),
}

// eciConfigMapManager implements ConfigMap Manager interface with
// simple operations to ECI.
type eciConfigMapManager struct {
	store   map[keyObject]*v1.ConfigMap
	locker  sync.RWMutex
	manager Manager
}

// NewEciConfigMapManager create a configmap manager for ECI.
func NewEciConfigMapManager() Manager {
	return ecicmm
}

// SetManager for configmap manager
func (m *eciConfigMapManager) SetManager(manager Manager) {
	m.locker.Lock()
	defer m.locker.Unlock()
	m.manager = manager
}

func (m *eciConfigMapManager) getConfigMapFromLocal(namespace, name string) (*v1.ConfigMap, error) {
	key := keyObject{
		namespace: namespace,
		name:      name,
	}
	configmap, ok := m.store[key]
	if !ok {
		return nil, api_errors.NewNotFound(v1.Resource("configmap"), fmt.Sprintf("%v/%v", namespace, name))
	}

	return configmap, nil
}

func (m *eciConfigMapManager) UpdateConfigMap(configMaps []v1.ConfigMap) {
	m.locker.Lock()
	defer m.locker.Unlock()

	for i := range configMaps {
		configMap := configMaps[i]
		key := keyObject{configMap.Namespace, configMap.Name}
		m.store[key] = &configMap
	}
}

func (m *eciConfigMapManager) DeleteConfigMap(configMaps []v1.ConfigMap) {
	m.locker.Lock()
	defer m.locker.Unlock()

	for i := range configMaps {
		configMap := configMaps[i]
		key := keyObject{configMap.Namespace, configMap.Name}
		delete(m.store, key)
	}
}

func (m *eciConfigMapManager) GetConfigMap(namespace, name string) (*v1.ConfigMap, error) {
	m.locker.RLock()
	defer m.locker.RUnlock()

	configMap, err := m.getConfigMapFromLocal(namespace, name)
	if api_errors.IsNotFound(err) && m.manager != nil {
		return m.manager.GetConfigMap(namespace, name)
	}
	return configMap, err
}

func (m *eciConfigMapManager) RegisterPod(pod *v1.Pod) {
	m.locker.Lock()
	defer m.locker.Unlock()

	if m.manager != nil {
		m.manager.RegisterPod(pod)
	}
}

func (m *eciConfigMapManager) UnregisterPod(pod *v1.Pod) {
	m.locker.Lock()
	defer m.locker.Unlock()

	if m.manager != nil {
		m.manager.UnregisterPod(pod)
	}
}

func (m *eciConfigMapManager) Clean() {
	m.locker.Lock()
	defer m.locker.Unlock()
	m.store = make(map[keyObject]*v1.ConfigMap)
}

func SetManager(manager Manager) {
	ecicmm.SetManager(manager)
}

func EciAddConfigMaps(configMaps []v1.ConfigMap) {
	ecicmm.UpdateConfigMap(configMaps)
}

func EciDeleteConfigMaps(configMaps []v1.ConfigMap) {
	ecicmm.DeleteConfigMap(configMaps)
}

func EciCleanConfigMaps() {
	ecicmm.Clean()
}
