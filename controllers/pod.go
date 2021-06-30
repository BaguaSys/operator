package controllers

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/BaguaSys/operator/api/v1alpha1"
)

// GetPodsForJob returns the pods managed by the job. This can be achieved by selecting pods using label key "job-name"
// i.e. all pods created by the job will come with label "job-name" = <this_job_name>
func (r *BaguaReconciler) GetPodsForJob(job interface{}) ([]*corev1.Pod, error) {
	obj, err := meta.Accessor(job)
	if err != nil {
		return nil, err
	}
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	podlist := &corev1.PodList{}
	err = r.List(context.Background(), podlist, client.MatchingLabels(r.JobController.GenLabels(obj.GetName())))
	if err != nil {
		return nil, err
	}

	return convertPodList(podlist.Items), nil
}

// convertPodList convert pod list to pod point list
func convertPodList(list []corev1.Pod) []*corev1.Pod {
	if list == nil {
		return nil
	}
	ret := make([]*corev1.Pod, 0, len(list))
	for i := range list {
		ret = append(ret, &list[i])
	}
	return ret
}

func (r *BaguaReconciler) GetDefaultContainerName() string {
	return api.DefaultContainerName
}

func (r *BaguaReconciler) GetDefaultContainerPortName() string {
	return ""
}

func (r *BaguaReconciler) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	rtype commonv1.ReplicaType, index int) bool {
	return false
}

// SetClusterSpec sets the cluster spec for the pod
func (r *BaguaReconciler) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	baguaJob := job.(*api.Bagua)

	if podTemplate.Spec.HostNetwork {
		podTemplate.Spec.DNSPolicy = corev1.DNSClusterFirstWithHostNet
	}

	podTemplate.Spec.Hostname = common.GenGeneralName(baguaJob.Name, rtype, index)
	podTemplate.Spec.Subdomain = common.GenGeneralName(baguaJob.Name, rtype, index)

	if !baguaJob.Spec.EnableElastic {
		return r.setClusterSpecUnderStaticMode(baguaJob, podTemplate, rtype, index)
	}
	return r.setClusterSpecUnderElasticMode(baguaJob, podTemplate, rtype, index)
}

func (r *BaguaReconciler) setClusterSpecUnderStaticMode(baguaJob *api.Bagua, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	if rtype != string(api.ReplicaMaster) && rtype != string(api.ReplicaWorker) {
		return nil
	}
	// rank
	rank, err := strconv.Atoi(index)
	if err != nil {
		return fmt.Errorf("cannot convert index %v from string to int", index)
	}
	if rtype == string(api.ReplicaWorker) {
		rank += 1
	}
	//master port
	var masterPort string
	portsMap, err := r.GetPortsFromJob(baguaJob.Spec.ReplicaSpecs[api.ReplicaMaster])
	if err != nil {
		return err
	}
	ports := []int64{}
	for _, p := range portsMap {
		ports = append(ports, int64(p))
	}
	sort.Slice(ports, func(i, j int) bool { return ports[i] < ports[j] })
	masterPort = strconv.FormatInt(ports[0], 10)

	//master address
	masterAddr := GetPodDomainName(baguaJob.Name, baguaJob.Namespace, string(api.ReplicaMaster), "0")
	if rtype == string(api.ReplicaMaster) {
		masterAddr = "0.0.0.0"
	}

	env := []corev1.EnvVar{
		{
			Name:  "MASTER_ADDR",
			Value: masterAddr,
		},
		{
			Name:  "MASTER_PORT",
			Value: masterPort,
		},
		{
			Name:  "RANK",
			Value: strconv.FormatInt(int64(rank), 10),
		},
		{
			Name:  "WORLD_SIZE",
			Value: strconv.FormatInt(int64(baguaJob.Spec.GetReplicas(api.ReplicaWorker)+1), 10),
		},
	}

	for i := range podTemplate.Spec.Containers {
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, env...)
	}

	if rtype == string(api.ReplicaWorker) {
		r.addInitContainer(baguaJob, podTemplate, rtype)
	}

	fixedOptions := strings.Join([]string{
		fmt.Sprintf("--nnodes=%v", baguaJob.Spec.GetReplicas(api.ReplicaWorker)+1),
		"--node_rank=$RANK",
		"--master_addr=$MASTER_ADDR",
		"--master_port=$MASTER_PORT",
	}, " ")
	hookCommand(podTemplate, fixedOptions)

	return nil
}

func (r *BaguaReconciler) setClusterSpecUnderElasticMode(baguaJob *api.Bagua, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	if rtype != string(api.ReplicaEtcd) && rtype != string(api.ReplicaWorker) {
		return nil
	}
	// etcd
	if rtype == string(api.ReplicaEtcd) {
		env := []corev1.EnvVar{
			{
				Name: "MY_NODE_IP",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "status.hostIP",
					},
				},
			},
		}
		for i := range podTemplate.Spec.Containers {
			podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, env...)
		}

		return nil
	}

	// worker
	etcdPortsMap, err := r.GetPortsFromJob(baguaJob.Spec.ReplicaSpecs[api.ReplicaEtcd])
	if err != nil {
		return err
	}

	etcdPorts := []int64{}
	for _, p := range etcdPortsMap {
		etcdPorts = append(etcdPorts, int64(p))
	}
	sort.Slice(etcdPorts, func(i, j int) bool { return etcdPorts[i] < etcdPorts[j] })

	etcdClientPort := etcdPorts[0]
	etcdHost := GetPodDomainName(baguaJob.Name, baguaJob.Namespace, string(api.ReplicaEtcd), "0")

	workerReplicas := baguaJob.Spec.GetReplicas(api.ReplicaWorker)
	minReplicas := workerReplicas
	maxReplicas := workerReplicas
	if baguaJob.Spec.MinReplicas != nil {
		minReplicas = *baguaJob.Spec.MinReplicas
	}
	if baguaJob.Spec.MaxReplicas != nil {
		maxReplicas = *baguaJob.Spec.MaxReplicas
	}

	r.addInitContainer(baguaJob, podTemplate, rtype)

	fixedOptions := strings.Join([]string{
		fmt.Sprintf("--nnodes=%v:%v", minReplicas, maxReplicas),
		fmt.Sprintf("--rdzv_id=%v", baguaJob.Name),
		fmt.Sprintf("--rdzv_backend=%v", "etcd-v2"),
		fmt.Sprintf("--rdzv_endpoint=%v:%v", etcdHost, etcdClientPort),
	}, " ")
	hookCommand(podTemplate, fixedOptions)

	return nil
}

func hookCommand(podTemplate *corev1.PodTemplateSpec, newOptions string) {
	l := len(podTemplate.Spec.Containers[0].Command)
	cmd := podTemplate.Spec.Containers[0].Command[l-1]
	cmdSlice := strings.Split(cmd, " ")
	for i, v := range cmdSlice {
		if strings.HasPrefix(v, "bagua.") {
			cmdSlice[i] += " " + newOptions
			break
		}
	}

	podTemplate.Spec.Containers[0].Command[l-1] = strings.Join(cmdSlice, " ")
}

func getFixedOptionsForMasterOrWorker(elastic bool, workerNum int32) string {
	if elastic {
		return strings.Join([]string{
			"--nnodes=$MIN_SIZE:$MAX_SIZE",
			"--rdzv_id=$JOB_ID",
			"--rdzv_backend=etcd-v2",
			"--rdzv_endpoint=$ETCD_HOST:$ETCD_CLIENT_PORT",
		}, " ")
	}
	return fmt.Sprintf("--nnodes=%v", workerNum+1) +
		" --node_rank=$RANK --master_addr=$MASTER_ADDR --master_port=$MASTER_PORT"
}

func (r *BaguaReconciler) addInitContainer(baguaJob *api.Bagua, podTemplate *corev1.PodTemplateSpec, rtype string) {
	initConatinerImage := r.Config.InitContainerImage

	var target commonv1.ReplicaType
	if !baguaJob.Spec.EnableElastic {
		target = api.ReplicaMaster
	} else {
		target = api.ReplicaEtcd
	}
	initContainer := getInitContainer(baguaJob.Name, baguaJob.Namespace, initConatinerImage, target)

	podTemplate.Spec.InitContainers = append(podTemplate.Spec.InitContainers, initContainer)
}

func getInitContainer(jobName, namespace, initImage string, target commonv1.ReplicaType) corev1.Container {
	host := GetPodDomainName(jobName, namespace, string(target), "0")
	return corev1.Container{
		Name:            "bagua-initializer",
		Image:           initImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"/bin/sh",
			"-c",
			fmt.Sprintf("until nslookup %v;do echo waiting for %v && sleep 5s; done", host, target),
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("50Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("200m"),
				corev1.ResourceMemory: resource.MustParse("100Mi"),
			},
		},
	}
}
