package status

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/util/flushwriter"
	"k8s.io/klog/v2"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
)

func getContainerNames(pod *v1.Pod) []string {
	cNames := make([]string, 0)

	for _, item := range pod.Spec.InitContainers {
		cNames = append(cNames, item.Name)
	}

	for _, item := range pod.Spec.Containers {
		cNames = append(cNames, item.Name)
	}

	return cNames
}

type LogBackupOptions struct {
	BackupDir  string
	TailLines  int64
	LimitBytes int64
}

func (m *manager) backupContainerLogs(pod *v1.Pod, options *LogBackupOptions, getLogsFunc GetContainerLogsFunc) error {
	startTime := time.Now()

	eciId, ok := pod.Annotations["k8s.cmecloud.cn/eci-instance-id"]
	if !ok {
		klog.Errorf("[backupContainerLogs] eciId not found in pod.Annotations")
		return errors.Errorf("eciId not found in pod.Annotations")
	}

	klog.Infof("[backupContainerLogs] backup container logs of eci %q starting ......", eciId)

	backupDir := options.BackupDir
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		klog.Errorf("[backupContainerLogs] mkdir %q failed: %s", backupDir, err)
		return err
	}

	for _, cName := range getContainerNames(pod) {
		ctrLogPath := path.Join(backupDir, fmt.Sprintf("%s_%s.log", eciId, cName))
		klog.Infof("[backupContainerLogs] backup container logs to %q starting ......", ctrLogPath)
		bakFile, err := os.OpenFile(ctrLogPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, os.ModeAppend|os.ModePerm)
		if err != nil {
			return errors.Wrapf(err, "open %q failed", ctrLogPath)
		}
		fw := flushwriter.Wrap(bakFile)
		logOptions := &v1.PodLogOptions{
			TypeMeta:   metav1.TypeMeta{},
			Container:  cName,
			Timestamps: true,
			TailLines:  &options.TailLines,
			LimitBytes: &options.LimitBytes,
		}
		if err := getLogsFunc(context.TODO(), kubecontainer.GetPodFullName(pod), cName, logOptions, fw, fw); err != nil {
			klog.Warningf("[backupContainerLogs] saveLogsToFile failed. ContainerName: %s, Error: %s", cName, err)
		}
		if err := bakFile.Close(); err != nil {
			klog.Warningf("[backupContainerLogs] close %q failed: %s", ctrLogPath, err)
		}
		klog.Infof("[backupContainerLogs] backup container logs to %q finished", ctrLogPath)
	}

	klog.Infof("[backupContainerLogs] backup container logs of eci %q finished. CostTime: %v", eciId, time.Since(startTime))

	return nil
}
