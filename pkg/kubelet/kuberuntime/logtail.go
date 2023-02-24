package kuberuntime

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/klog/v2"
)

// StaticContainer for logtail
type StaticContainer struct {
	ID       string            `json:"ID,omitempty"`
	Name     string            `json:"Name,omitempty"`
	HostName string            `json:"HostName,omitempty"`
	IP       string            `json:"IP,omitempty"`
	Image    string            `json:"Image,omitempty"`
	LogPath  string            `json:"LogPath,omitempty"`
	Labels   map[string]string `json:"Labels,omitempty"`
	LogType  string            `json:"LogType,omitempty"`
	Env      map[string]string `json:"Env,omitempty"`
	Mounts   []Mount           `json:"Mounts,omitempty"`
	UpperDir string            `json:"UpperDir,omitempty"`
}

// Mount for logtail
type Mount struct {
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	Driver      string `json:"Driver"`
}

const (
	staticHostPath = "/etc/eci-agent/ilogtail/"
	staticHostFile = staticHostPath + "container.json"
)

//no lock here, for single pod, start container one by one
//no need rm pod from containerMap, because containerMap only keep one pod now
var containerMap = make(map[types.UID]map[string]StaticContainer)

func init() {
	loadStaticContainer()
}

func loadStaticContainer() {
	cf, err := ioutil.ReadFile(staticHostFile)
	if err != nil {
		return
	}
	var staticContainers []StaticContainer
	err = json.Unmarshal(cf, &staticContainers)
	if err != nil || len(staticContainers) == 0 {
		return
	}
	pID := staticContainers[0].Labels["io.kubernetes.pod.uid"]
	if len(pID) == 0 {
		return
	}
	podUID := types.UID(pID)

	containerMap[(podUID)] = make(map[string]StaticContainer)
	for _, c := range staticContainers {
		containerMap[podUID][c.Name] = c
	}
}

func WriteLogtailStaticContainerfile(pod *v1.Pod, sandboxConfig *runtimeapi.PodSandboxConfig, containerName, containerID string, restartCount int) error {

	//update container map
	updateContainerMap(pod, sandboxConfig, containerName, containerID, restartCount)
	return writefile(pod)
}

func updateContainerMap(pod *v1.Pod, sandboxConfig *runtimeapi.PodSandboxConfig, containerName, containerID string, restartCount int) {
	pcm, ok := containerMap[pod.UID]
	if !ok {
		pcm = make(map[string]StaticContainer)
		containerMap[pod.UID] = pcm
	}

	emptyDirs := make(map[string]interface{})
	for _, volume := range pod.Spec.Volumes {
		if volume.EmptyDir != nil {
			emptyDirs[volume.Name] = nil
		}
	}

	containers := append(pod.Spec.Containers, pod.Spec.InitContainers...)

	pcm["POD"] = StaticContainer{
		ID:       sandboxConfig.Metadata.Uid,
		Name:     "POD",
		Labels:   sandboxConfig.Labels,
		Image:    "pause:latest",
		LogPath:  sandboxConfig.GetLogDirectory(),
		LogType:  "json-file",
		UpperDir: "/run/containerd/io.containerd.runtime.v2.task/k8s.io/null/rootfs",
	}

	for _, c := range containers {
		if c.Name == containerName {
			sc := StaticContainer{
				ID:    containerID,
				Name:  c.Name,
				Image: c.Image,
				//IP:       ip,
				//HostName: hostname,
				UpperDir: fmt.Sprintf("/run/containerd/io.containerd.runtime.v2.task/k8s.io/%s/rootfs", containerID),
				Env:      envToMap(c.Env),
				LogPath:  path.Join(BuildContainerLogsDirectory(pod.Namespace, pod.Name, pod.UID, c.Name), fmt.Sprintf("%d.log", restartCount)),
				LogType:  "json-file",
			}
			sc.Labels = map[string]string{
				"io.kubernetes.container.name": c.Name,
				"io.kubernetes.pod.name":       pod.GetName(),
				"io.kubernetes.pod.namespace":  pod.GetNamespace(),
				"io.kubernetes.pod.uid":        string(pod.UID),
			}
			for _, volumeMount := range c.VolumeMounts {
				if _, ok := emptyDirs[volumeMount.Name]; !ok {
					continue
				}
				sc.Mounts = append(sc.Mounts, Mount{
					Source:      fmt.Sprintf("/var/lib/kubelet/pods/%s/volumes/kubernetes.io~empty-dir/%s", pod.UID, volumeMount.Name),
					Destination: volumeMount.MountPath,
				})
			}
			pcm[c.Name] = sc
		}
	}
}

//generate static container file
func writefile(pod *v1.Pod) error {
	var staticContainers []StaticContainer

	for _, c := range containerMap[pod.UID] {
		staticContainers = append(staticContainers, c)
	}

	os.MkdirAll(staticHostPath, 0755)
	bsc, _ := json.MarshalIndent(staticContainers, "", "	")

	if err := SyncWrite(staticHostFile, bsc); err != nil {
		klog.Error("write ilogtail static container file to %q failed", staticHostFile)
		return fmt.Errorf("write ilogtail static container file to %q failed", staticHostFile)
	}
	return nil
}

func envToMap(envs []v1.EnvVar) map[string]string {
	em := make(map[string]string)
	for _, env := range envs {
		em[env.Name] = env.Value
	}
	return em
}

func SyncWrite(file string, data []byte) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err != nil {
		return err
	}
	return f.Sync()
}
