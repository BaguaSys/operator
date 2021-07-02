package v1alpha1

import (
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
)

const (
	ErrInvalidDefinition = "InvalidDefinition"
	JobOwnerKey          = ".metadata.controller"
	DefaultContainerName = "bagua"
)

const (
	ReplicaMaster commonv1.ReplicaType = "master"
	ReplicaWorker commonv1.ReplicaType = "worker"
	ReplicaEtcd   commonv1.ReplicaType = "etcd"
)
