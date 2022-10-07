package pkg

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"os"
	"path/filepath"
)

// InitKubernetesCli 创建 k8s 客户端，与 apiserver 交互
func InitKubernetesCli() (*kubernetes.Clientset, error) {
	var (
		err    error
		config *rest.Config
	)

	// 部署在集群内  不需要 kubeconfig 模式
	// https://blog.51cto.com/u_10983441/4976185
	// 当程序以pod方式运行时，就直接走这里的逻辑
	// rest.InClusterConfig 直接使用pod中自带的token等内容
	//if config, err = rest.InClusterConfig(); err != nil {
	//	return nil, err
	//}
	if config, err = CreateKubeConfig(); err != nil {
		return nil, err
	}
	// 创建 ClientSet 对象
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	log.Println("成功获取kubeconfig")
	return clientset, nil
}

func CreateKubeConfig() (*rest.Config, error) {
	kubeConfigPath := ""
	if home := homedir.HomeDir(); home != "" {
		kubeConfigPath = filepath.Join(home, ".kube", "config")
	}
	//home := "/Users/dufengyang/"
	//kubeConfigPath = filepath.Join(home, ".kube", "config")
	fileExist, err := PathExists(kubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("justify kubeConfigPath exist err,err:%v", err)
	}
	//.kube/config文件存在，就使用文件
	//这里主要是本地测试
	if fileExist {
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return nil, err
		}
		return config, nil
	} else {
		//当程序以pod方式运行时，就直接走这里的逻辑
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		return config, nil
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
