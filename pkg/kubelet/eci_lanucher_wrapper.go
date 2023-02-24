package kubelet

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/configmap"
	"k8s.io/kubernetes/pkg/kubelet/secret"
	volumetypes "k8s.io/kubernetes/pkg/volume/util/types"
)

// AddSecrets add or update secret.
func (kl *Kubelet) AddSecrets(secrets []v1.Secret) {
	secret.EciAddSecrets(secrets)
}

func (kl *Kubelet) DeleteSecrets(secrets []v1.Secret) {
	secret.EciDeleteSecrets(secrets)
}

// GetSecret return the specified secret. only for multi eth mode.
func (kl *Kubelet) GetSecret(namespace, name string) (*v1.Secret, error) {
	return secret.EciGetSecret(namespace, name)
}

// CleanSecrets clean all Secrets in manager.
func (kl *Kubelet) CleanSecrets() {
	secret.EciCleanSecrets()
}

// AddConfigMaps add or update ConfigMap
func (kl *Kubelet) AddConfigMaps(configMaps []v1.ConfigMap) {
	configmap.EciAddConfigMaps(configMaps)
}

func (kl *Kubelet) DeleteConfigMaps(configMaps []v1.ConfigMap) {
	configmap.EciDeleteConfigMaps(configMaps)
}

// CleanConfigMaps clean all ConfigMaps in manager.
func (kl *Kubelet) CleanConfigMaps() {
	configmap.EciCleanConfigMaps()
}

func (kl *Kubelet) GetPushQueue() <-chan v1.PodStatus {
	return kl.statusManager.GetPushQueue()
}

func (kl *Kubelet) GetPodStatus(pod *v1.Pod) *v1.PodStatus {
	if status, ok := kl.statusManager.GetPodStatus(pod.UID); ok {
		return &status
	}

	return nil
}

// Trigger container garbage collect.
func (kl *Kubelet) ContainerGarbageCollect() error {
	return kl.containerGC.GarbageCollect()
}

func (kl *Kubelet) MarkVolumeFSRequiredResize(podName volumetypes.UniquePodName, volumeName v1.UniqueVolumeName) error {
	return kl.volumeManager.MarkVolumeFSRequireResize(volumeName, podName)
}

func (kl *Kubelet) GetVolumeResizeStatus(podName volumetypes.UniquePodName, volumeName v1.UniqueVolumeName) (bool, error) {
	return kl.volumeManager.GetVolumeResizeStatus(volumeName, podName)
}
