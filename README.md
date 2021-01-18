# 自定义 operator 试用

## 安装

安装：

> https://sdk.operatorframework.io/docs/installation/

## 试用

参考官方示例：https://sdk.operatorframework.io/docs/building-operators/golang/

我的环境信息:

* operator-sdk version v1.3.0
* git version 1.8.3.1
* go version go1.15.4 linux/amd64
* mercurial version 5.6.1-1.x86_64
* docker version 19.03.0
* kubectl version v1.18.3
* kubernetes cluster v1.18.3

``` BASH
# 初始化
$ operator-sdk init --domain=jackzhang.io --repo=github.com/JackZxj/operator-demo

# 查看帮助：
# $ operator-sdk create api --help
# 创建一个api,，分组为demo，版本为v1alpha1，crd类型为OperatorTester
$ operator-sdk create api --group demo --version v1alpha1 --kind OperatorTester
# 查看生成的样例
$ cat config/samples/demo_v1alpha1_operatortester.yaml

# 编辑 api/v1alpha1/operatortester_types.go 修改crd定义
# 编辑 controllers/operatortester_controller.go 修改控制器
# 生成crd清单
$ make generate
# 指定权限并生成RBAC清单
$ make manifests
# 构建并运行
$ make install
# 修改baseimage dockerfile以构建镜像
# 构建镜像(如果没有写test,可以注释掉docker-build里的test项,直接build)
$ make docker-build IMG=172.31.0.7:5000/operator-demo:v0.0.1
# 推到镜像仓库
$ make docker-push IMG=172.31.0.7:5000/operator-demo:v0.0.1
# 安装部署(config/default/manager_auth_proxy_patch.yaml中的镜像可能拉不下来)
$ make deploy IMG=172.31.0.7:5000/operator-demo:v0.0.1
# 根据定义的type创建yaml,然后创建实例
$ kubectl apply -f config/samples/demo_v1alpha1_operatortester.yaml
# 查看部署结果
$ kubectl get deploy -w
# 查看日志
$ kubectl logs operator-demo-controller-manager-c9b7dc9bc-b6lnv -c manager --tail 30 -n operator-demo-system
# 删除实例
$ kubectl delete -f config/samples/demo_v1alpha1_operatortester.yaml
# 删除部署
$ make undeploy
```

```BASH
# 构建source和destination两端的镜像，并推送到仓库
cd deployment/source
docker build -t 172.31.0.7:5000/source:v0.0.3 .
docker push 172.31.0.7:5000/source:v0.0.3
cd ../destination
docker build -t 172.31.0.7:5000/destination:v0.0.2 .
docker push 172.31.0.7:5000/destination:v0.0.2


# 构建镜像(如果没有写test,可以注释掉docker-build里的test项,直接build)
make docker-build IMG=172.31.0.7:5000/operator-demo:v0.1.0
# 推到镜像仓库
make docker-push IMG=172.31.0.7:5000/operator-demo:v0.1.0
# 安装部署(config/default/manager_auth_proxy_patch.yaml中的镜像可能拉不下来)
make deploy IMG=172.31.0.7:5000/operator-demo:v0.1.0
kubectl get deploy -n operator-demo-system -o wide
# 根据定义的type创建yaml,然后创建实例
kubectl apply -f config/samples/demo_v1alpha1_operatortester.yaml
# 查看
kubectl get OperatorTester operatortester-sample
# 删除实例
kubectl delete -f config/samples/demo_v1alpha1_operatortester.yaml
```