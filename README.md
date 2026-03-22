# Учебный проект

## Зависимости

1. minikube: `brew install minikube`
2. [goose](https://github.com/pressly/goose) для миграций: `brew install goose`
2. istioctl: `brew install istioctl`

## Запуск

1. Запустить кластер миникуба `make minikube-up`
2. Настроить k8s `make apply` - скорее всего придется два раза повторить, так как файлы применяются в случайном порядке и могут
быть ошибки с неймспейсами
3. Создать топик в кафке `make create-topic-product-management.product`
4. Задеплоить сервисы `make release`
5. Поднять туннель до миникуба: `minikube tunnel`. Запросы можно будет слать по `127.0.0.1:80`, если адрес не работает, то его можно посмотреть в EXTERNAL-IP в выводе команды `kubectl get svc istio-ingressgateway -n istio-system`

На шагах 3-4 могут быть ошибки, если сервисы еще не успели подняться. Достаточно повторить команду чуть позже.

Примеры запросов:
```
curl localhost/product -d '{"name":"test","price":100}' -H 'X-User-Id: 1'
curl localhost/products/search
```
