package v2Util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
)

func ParseExecConfigJson(k8sExecConfigJson *string) (*kubernetesplatform.KubernetesExecutorConfig, error) {
	var k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
	if *k8sExecConfigJson != "" {
		glog.Infof("input kubernetesConfig:%s\n", PrettyPrint(*k8sExecConfigJson))
		k8sExecCfg = &kubernetesplatform.KubernetesExecutorConfig{}
		if err := util.UnmarshalString(*k8sExecConfigJson, k8sExecCfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Kubernetes config, error: %w\nKubernetesConfig: %v", err, k8sExecConfigJson)
		}
	}
	return k8sExecCfg, nil
}

// TODO: use this everywhere in cli driver?
func PrettyPrint(jsonStr string) string {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, []byte(jsonStr), "", "  ")
	if err != nil {
		return jsonStr
	}
	return prettyJSON.String()
}
