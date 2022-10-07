package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"github.com/oceanweave/admission-webhook-sample/pkg"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"math/big"
	"os"
	"time"
)

func main() {
	log.Println("Start TLS build: ", "创建证书")
	// CA 配置
	subject := pkix.Name{
		Country:            []string{"CN"},
		Province:           []string{"Beijing"},
		Locality:           []string{"Beijing"},
		Organization:       []string{"dfy.io"},
		OrganizationalUnit: []string{"dfy.io"},
	}
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2021),
		// 注册信息
		Subject: subject,
		// 有效期  10 年有效期
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(10, 0, 0),
		// 根证书
		IsCA: true,
		// 可以用于  客户端认证 和 服务端认证
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// 生成 CA 私钥  4096 位的加密
	caPrivkey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Panic(err)
	}

	// 创建自签名的 CA 证书
	// 其中两个 相同的 ca 就是表示 自签
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivkey.PublicKey, caPrivkey)
	if err != nil {
		log.Panic(err)
	}

	// 编码证书文件  ca机构证书 用于验证其他证书
	caPEM := new(bytes.Buffer)
	if err := pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	}); err != nil {
		log.Panic(err)
	}

	// 对哪些服务进行签名，下面其实值的是同一个服务
	dnsName := []string{"admission-registry",
		"admission-registry.default",
		"admission-registry.default.svc",
		"admission-registry.default.svc.cluster.local",
	}
	commonName := "admission-registry.default.svc"
	subject.CommonName = commonName
	// 服务端的证书配置
	cert := &x509.Certificate{
		DNSNames:     dnsName,
		SerialNumber: big.NewInt(2020),
		Subject:      subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// 生成服务端的私钥
	serverPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Panic(err)
	}

	// 对服务端私钥进行签名
	// 使用 ca 证书对 服务端证书 进行加密
	serverCertBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &serverPrivKey.PublicKey, caPrivkey)
	if err != nil {
		log.Panic(err)
	}
	// 生成 pem 格式的证书
	serverCertPEM := new(bytes.Buffer)
	if err := pem.Encode(serverCertPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: serverCertBytes,
	}); err != nil {
		log.Panic(err)
	}

	// 还有 server 端的私钥
	serverPrivKeyPEM := new(bytes.Buffer)
	// 复制代码 误写为 serverCertPEM 要注意 ！！！
	if err := pem.Encode(serverPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(serverPrivKey),
	}); err != nil {
		log.Panic(err)
	}

	// 已经生成了 CA server.pem(公钥） server-key.pem（私钥）
	// 注意此处的权限配置，之前配置 0666  无法创建 ，配置 0766 才有权限创建所有子目录
	if err := os.MkdirAll("./etc/webhook/certs/", 0766); err != nil {
		log.Panic(err)
	}

	// 记录 ca 机构证书
	if err := WriteFile("./etc/webhook/certs/ca.crt", caPEM.Bytes()); err != nil {
		log.Panic(err)
	}

	if err := WriteFile("./etc/webhook/certs/tls.crt", serverCertPEM.Bytes()); err != nil {
		log.Panic(err)
	}

	if err := WriteFile("./etc/webhook/certs/tls.key", serverPrivKeyPEM.Bytes()); err != nil {
		log.Panic(err)
	}

	log.Println("webhook server tls generated successfully")
	log.Println("End TLS build: ", "证书创建完成")
	// 创建 ValidatingWebhookConfiguration  MutatingWebhookConfiguration
	if err := CreateAdmissionConfig(caPEM); err != nil {
		log.Panic(err)
	}

}

// WriteFile 将证书文件保存到 指定路径的文件中
func WriteFile(filePath string, bts []byte) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(bts); err != nil {
		return err
	}

	return nil
}

func CreateAdmissionConfig(caCert *bytes.Buffer) error {
	clientset, err := pkg.InitKubernetesCli()
	if err != nil {
		return err
	}
	// 通过定义环境变量，传入信息，来创建相应的 validate 和 mutate webhook
	var (
		webhookNamespace, _ = os.LookupEnv("WEBHOOK_NAMESPACE")
		validateCfgName, _  = os.LookupEnv("VALIDATE_CONFIG")
		mutateCfgName, _    = os.LookupEnv("MUTATE_CONFIG")
		webhookService, _   = os.LookupEnv("WEBHOOK_SERVICE")
		validatePath, _     = os.LookupEnv("VALIDATE_PATH")
		mutatePath, _       = os.LookupEnv("MUTATE_PATH")
	)
	log.Println("读取环境变量 validateCfgName ：", validateCfgName)
	ctx := context.Background()
	if validateCfgName != "" {
		// 创建 ValidatingWebhookConfiguration
		// api 包保存了基础资源的 数据结构
		// 下面的复制就对应了 ValidatingWebhookConfiguration 的yaml文件结构
		validateConfig := &admissionv1.ValidatingWebhookConfiguration{
			// apimachinery 的 meta1 是基础架构
			ObjectMeta: metav1.ObjectMeta{
				Name: validateCfgName,
			},
			Webhooks: []admissionv1.ValidatingWebhook{
				{
					// 可以创建多个 validate webhook，这是其中一个的名字
					Name: "io.dfy.admission-registry",
					ClientConfig: admissionv1.WebhookClientConfig{
						// ca 机构证书，相当于 ca 公钥，用于验证服务端穿来的证书
						// 此处是将 ca 证书传给 apiserver
						CABundle: caCert.Bytes(),
						Service: &admissionv1.ServiceReference{
							// 创建的 webhook service 名字
							Name:      webhookService,
							Namespace: webhookNamespace,
							// 访问路径
							Path: &validatePath,
						},
					},
					// 作用于哪些资源，就是监控到这些资源的变化，pod 的创建，进行准入控制
					Rules: []admissionv1.RuleWithOperations{
						{
							Operations: []admissionv1.OperationType{admissionv1.Create},
							Rule: admissionv1.Rule{
								APIGroups:   []string{""},
								APIVersions: []string{"v1"},
								Resources:   []string{"pods"},
							},
						},
					},
					AdmissionReviewVersions: []string{"v1"},
					// 注意此处的类型转换，直接取地址& 取不到，因此做了如下转换
					SideEffects: func() *admissionv1.SideEffectClass {
						se := admissionv1.SideEffectClassNone
						return &se
					}(),
				},
			},
		}
		// 客户端的创建是通过 client-go 包
		validateAdmissionClient := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations()
		_, err := validateAdmissionClient.Get(ctx, validateCfgName, metav1.GetOptions{})
		if err != nil {
			// 没找到 就创建 ValidatingWebhookConfiguration
			if errors.IsNotFound(err) {
				if _, err := validateAdmissionClient.Create(ctx, validateConfig, metav1.CreateOptions{}); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	if mutateCfgName != "" {
		// 创建 MutatingWebhookConfiguration
		mutateConfig := &admissionv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: mutateCfgName,
			},
			Webhooks: []admissionv1.MutatingWebhook{
				{
					Name: "io.dfy.admission-registry-mutate",
					ClientConfig: admissionv1.WebhookClientConfig{
						// ca 机构证书  公钥
						CABundle: caCert.Bytes(),
						Service: &admissionv1.ServiceReference{
							Name:      webhookService,
							Namespace: webhookNamespace,
							Path:      &mutatePath,
						},
					},
					Rules: []admissionv1.RuleWithOperations{
						{
							Operations: []admissionv1.OperationType{admissionv1.Create},
							Rule: admissionv1.Rule{
								APIGroups:   []string{"apps", ""},
								APIVersions: []string{"v1"},
								Resources:   []string{"deployments", "services"},
							},
						},
					},
					AdmissionReviewVersions: []string{"v1"},
					SideEffects: func() *admissionv1.SideEffectClass {
						se := admissionv1.SideEffectClassNone
						return &se
					}(),
				},
			},
		}
		mutateAdmissionClient := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations()
		_, err := mutateAdmissionClient.Get(ctx, validateCfgName, metav1.GetOptions{})
		if err != nil {
			// 没找到 就创建 ValidatingWebhookConfiguration
			if errors.IsNotFound(err) {
				if _, err := mutateAdmissionClient.Create(ctx, mutateConfig, metav1.CreateOptions{}); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}
	return nil
}
