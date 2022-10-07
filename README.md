# admission-webhook-sample
## 参考

- [实现一个容器镜像白名单的准入控制器 | 视频文字稿](https://mp.weixin.qq.com/s?__biz=MzU4MjQ0MTU4Ng==&mid=2247489670&idx=1&sn=37a8183c74f59d44e0e3183e41710a66&chksm=fdb9179bcace9e8dfb4d8a73404a83affec0de9b9269305c8fc9d9ae6354cdabdfa760794a29&scene=21#wechat_redirect)
- [自动管理 Admission Webhook TLS 证书](https://jishuin.proginn.com/p/763bfbd3890c)
- AdmissionReview 的完整结构定义：https://github.com/kubernetes/api/blob/master/admission/v1/types.go
- k8s api 开发手册：https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/

## 项目说明

``` sh
.
├── Dockerfile        # 多阶段构建
├── README.md
├── debug-sample      # 调试过程中创建的资源，可以忽略
│   ├── deploy.yaml
│   ├── service.yaml
│   ├── test-tls-pod.yaml
│   ├── test-webhook-pod.yaml
│   └── validata.yaml
├── deploy            # 部署和删除
│   ├── auto-deploy.yaml
│   └── uninstall.sh
├── download-deps.sh  # 拉取 k8s.io/kubernetes依赖
├── etc 				      # 测试时候使用的  可以忽略
│   └── webhook
│       └── certs
│           ├── tls.crt
│           └── tls.key
├── go.mod            # 依赖
├── go.sum
├── pkg								# 业务区
│   ├── utils.go			# 连接 k8s 集群
│   └── webhook.go		# webhook server 端，负责处理逻辑，实现准入控制，根据 admissionreview 返回 response
├── test-sample       # 部署后的测试，在白名单和不在白名单中的 pod ，以及给 service 添加 annotation 测试等
│   ├── test-deploy1.yaml
│   ├── test-deploy2.yaml
│   ├── test-pod.yaml
│   └── test-pod1.yaml
├── tls               # initContainer 使用，创建 CA 机构和 Server 端证书，同时用 CA 机构证书构建 validate 和 mutate 资源
│   └── main.go
└── webhook           # 连接 k8s 集群，根据请求的路径，调用后端处理程序，就是 pkg 目录下的 webhook.go
    └── main.go

```

## 注意

### 1. 使用后删除 validatingwebhookconfigurations  和 mutatingwebhookconfigurations

> 这两个资源会在 pod 、 deployment、service 创建时校验，若不删除，会影响这些资源的创建
>
> - validatingwebhookconfigurations  和 mutatingwebhookconfigurations 只是将【关注资源的请求】转发到【对应服务的对应路径上】
> - 若不删除这些资源，仍会转发到【对应的admission服务】，但此时服务已经删除，就会导致【卡在此处，无法继续创建，同时可能看不到报错信息】

``` sh
#! /bin/sh

kubectl delete -f auto-deploy.yaml
kubectl delete validatingwebhookconfigurations.admissionregistration.k8s.io admission-registry
kubectl delete mutatingwebhookconfigurations.admissionregistration.k8s.io admission-registry-mutate
```



``` yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  creationTimestamp: "2022-10-07T01:57:10Z"
  generation: 1
  name: admission-registry
  resourceVersion: "32755"
  uid: b6058e85-b9d3-4f18-9ae5-a9770c384079
webhooks:
- admissionReviewVersions:
  - v1
  clientConfig:
    caBundle: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUZnekNDQTJ1Z0F3SUJBZ0lDQitVd0RRWUpLb1pJaHZjTkFRRUxCUUF3VXpFTE1Ba0dBMVVFQmhNQ1EwNHgKRURBT0JnTlZCQWdUQjBKbGFXcHBibWN4RURBT0JnTlZCQWNUQjBKbGFXcHBibWN4RHpBTkJnTlZCQW9UQm1SbQplUzVwYnpFUE1BMEdBMVVFQ3hNR1pHWjVMbWx2TUI0WERUSXlNVEF3TnpBeE5UY3dPRm9YRFRNeU1UQXdOekF4Ck5UY3dPRm93VXpFTE1Ba0dBMVVFQmhNQ1EwNHhFREFPQmdOVkJBZ1RCMEpsYVdwcGJtY3hFREFPQmdOVkJBY1QKQjBKbGFXcHBibWN4RHpBTkJnTlZCQW9UQm1SbWVTNXBiekVQTUEwR0ExVUVDeE1HWkdaNUxtbHZNSUlDSWpBTgpCZ2txaGtpRzl3MEJBUUVGQUFPQ0FnOEFNSUlDQ2dLQ0FnRUF1eGpDMElLSDNVdklvV1pBZTROY2pOSlphd1BoCm5Pc0Q5Y09xQzV1V0tnQlBpamdxWXB2bUJTamcrWVh0MXRDMk00WEI1aDlQdVJaV01KdW9ZcTlhVnZpTC9pM2YKa3Myck1leStKMkNMY1ZvcmdlNC82SHVXNy9VSUtMTlhrbkNLK0FZUis5dDVkTVFHZHR6YXpsNlBXS0IvUlpvSgpyNk0yODdHQU1vNW1zcnZ0WEN3WDhKOTlRKzVWd3lGb20veFZmR0dtMHU0dXBRQkl4VzVUM2JuVE5CSlc2eVZhCkt4VmRwMFVmb0hTczRNem1QVHJlbm5MQ2dzZmpsbkhUa1YxNGo0d3hyZXVVQkhrZmZHcjI1V0NDTlN0eGRSazMKaXRiZ2FtSFppa3U4bTdGTlg4SlJ3Qml0VDZ2cHg4aXFaSURybEFucW5vbUxVZWFRdW9jUTZhRWtWS3A0UDFIOApQSlg5bnlXRlRpREt3Q3ZSVVJXRTVKZm1wcmJPRElnakdnczQ4V2NnMTFXWDBLOW1pN05YQ3dKQm1NTmFvU09pCjZIVlRqaDMwV2Q1TCtPM0p4T3hjUHc0T2hPSWowL3FmR0VkelpyZXFnSTBVUm9STW8zY2JZNFZiSGpNM1VDNWoKOEdETVo0VWVZNnMrWnVWSTlKclJDSE5zVkdZNVdlcWV0SXBwWXhpcGxoQ1RmbTFFVUllYUZ3YUVLSjBGVjZhaQp1QzIzTWZEYnRWZng0UWxCRkZ0R252bjBoTUNVWDFiV2ZHZDNxMXF0UEZ1VE9yMnRoUno5Wk5nQmRtUHdhT3ZjCnFLdGdmMGRhRm5vVkM2dUMxc0tOL2JnNDZ3Y1BBcm1hY1Q4dXR1c1Z4a3dOOW1jUkFwQVpnQUVFRUMzaDJoVGkKNXl0cjRBVUVVWDIyZGo4Q0F3RUFBYU5oTUY4d0RnWURWUjBQQVFIL0JBUURBZ0tFTUIwR0ExVWRKUVFXTUJRRwpDQ3NHQVFVRkJ3TUNCZ2dyQmdFRkJRY0RBVEFQQmdOVkhSTUJBZjhFQlRBREFRSC9NQjBHQTFVZERnUVdCQlRWCk1yOVJJWXJYeHRvakZjVkFVbTJMbzNjQWtUQU5CZ2txaGtpRzl3MEJBUXNGQUFPQ0FnRUFUdFo2UHhYYVI2Z28Kc2lIYk5tdmszNTg0WlEzT0ZURnJoYzJ2ak9ONnN1TElnSWd5M3RFQ3NjRks5VzNRRFN5RWVNMWdCNFpRQ2pldAo4N0ZETVhZblF1djFzdmpkNlk1ajF3VUJPZnErZjI0clFLaEg4UzdyS0F5SjZNdkgzU3gybUtJaWh4eFpFZHR4CnBaS3AwV3Z4Wkgxb1hNL2hhL2RQQWx6UmdjbG5HcmQ5dGgyYm9vNUhKSXJvZy9uUG5PSDE2SzlhdHdSYVgySXgKbldFQTdwWk01dU9HYzJMVWlHdXJ5Zm15VEhuMGZDS3I1ckJJRkwyVWU2eGlGRHlKR1Y4ZXdEazhSWmdQcWZMSgpZellVYkFvdTFXVWRRU2hWL2FUWElDdHo4SHc4blFHTTdtT1JNRkphOGQ5dE42RHBBcDZPOVk3WHkyUkhuK1lwClR2SlpYUEdpQUhnOFhlOFNHcmpUeHZCeit3ZlBQcjVOK1V6cWx4S3c1OU9HUUtGSHUwcjdsTUcxQXIrK09tVVEKd3lDYnphdVNsUU5vN0ZoUG54TG5NUzlWbTNpazJ5SmNuUC9rMXF1Y2d2L055cUdEcHJPZUsxRXlWMWRiVUpiZApDRWtGenoycHdWbjM5aStuOTBVa0NsLzRsWEtPckYxZ3locGFhL2s0VHc1UHlWQ2o2OWtIZ3JhYjRzVjVST2x0CnYxcE5EcTl6bWd0TVUyOTk0T3o5YUtBanlsOUphWC9EbGVlSHhNeEw1clFuMzhTaks1TUplOXcwb0FjVHptT2sKUW5tVURlemRTaGlSOTRtbk9ycW42MHRHbCt1ZHNBdExuM2srOGtUTVBEM1o1a3BPUWNERTQ4WUxJdzlEQXhlQwpUbHZld1g3UTJQZFRVanQ5MjR1cytUZkFVcXY0TDFnPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
    service:
      name: admission-registry     # 会转发到该服务 admission-registry 下的 /validate 路径，也就是调用 webhook 对应的处理程序
      namespace: default
      path: /validate
      port: 443
  failurePolicy: Fail
  matchPolicy: Equivalent
  name: io.dfy.admission-registry  # 此 validate 的名称，可以配置多个
  namespaceSelector: {}
  objectSelector: {}
  rules:   												 # 此 validate 关注 pod 的创建
  - apiGroups:
    - ""
    apiVersions:
    - v1
    operations:
    - CREATE
    resources:
    - pods
    scope: '*'
  sideEffects: None
  timeoutSeconds: 10
```



### 2. 白名单控制

> 目前在 webhook 中实现的逻辑是
>
> 1. validate 作用是校验：
     >
     >    - 校验镜像的来源是否是白名单内，不在白名单内就阻断 pod 创建
>
>    - 通过 WHITELIST_REGISTRIES 指定镜像白名单的配置
>
> 2. mutate 作用是修改：
     >
     >    - 目前是校验 io.ydzs.admission-registry/mutate=no/off/false/n 是否具有此 annotation
>    - 若没有，就添加此 annotation ：io.ydzs.admission-registry/status=mutated

``` yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: admission-registry-sa

---

apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: admission-registry-clusterrole
rules:
  - verbs: ["*"]
    resources: ["validatingwebhookconfigurations", "mutatingwebhookconfigurations"]
    apiGroups: ["admissionregistration.k8s.io"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: admission-registry-clusterrolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: admission-registry-clusterrole
subjects:
  - kind: ServiceAccount
    name: admission-registry-sa
    namespace: default

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: admission-registry
  labels:
    app: admission-registry
spec:
  selector:
    matchLabels:
      app: admission-registry
  template:
    metadata:
      labels:
        app: admission-registry
    spec:
      serviceAccountName: admission-registry-sa
      initContainers:
        - name: webhook-init
          image: dfy007/admission-registry-tls:v1.8
          imagePullPolicy: IfNotPresent
          env:
            - name: WEBHOOK_NAMESPACE
              value: default
            - name: WEBHOOK_SERVICE
              value: admission-registry
            - name: VALIDATE_CONFIG
              value: admission-registry
            - name: VALIDATE_PATH
              value: /validate
            - name: MUTATE_CONFIG
              value: admission-registry-mutate
            - name: MUTATE_PATH
              value: /mutate
          volumeMounts:
            - name: webhook-certs
              mountPath: /etc/webhook/certs
      containers:
        - name: webhook
          image: dfy007/admission-webhook-example:v1.8
          imagePullPolicy: IfNotPresent
          env:
            - name: WHITELIST_REGISTRIES  # 更改白名单的控制
              value: "docker.io,gcr.io"
          ports:
            - containerPort: 443
          volumeMounts:
            - name: webhook-certs
              mountPath: /etc/webhook/certs
              readOnly: true
      volumes:
        - name: webhook-certs
          emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: admission-registry
  labels:
    app: admission-registry
spec:
  ports:
    - port: 443
      targetPort: 443
  selector:
    app: admission-registry
```

