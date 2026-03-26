# Учебный проект

Содержит настройки для minikube в папке k8s/ и два микросервиса на go в папке backend/.

## Зависимости

1. minikube: `brew install minikube`
2. [goose](https://github.com/pressly/goose) для миграций: `brew install goose`
2. istioctl: `brew install istioctl`

## Запуск

1. Запустить кластер миникуба `make minikube-up`
2. Настроить k8s `make apply`
3. Создать топик в кафке `make create-topic-product-management.product`
4. Создать маппинги в elastic: `make create-mappings-products`
5. Задеплоить сервисы `make release`
6. Поднять туннель до миникуба: `minikube tunnel`. Запросы можно будет слать по `127.0.0.1:80`, если адрес не работает, то его можно посмотреть в EXTERNAL-IP в выводе команды `kubectl get svc istio-ingressgateway -n istio-system`
7. Посмотреть логи сервисов: `make logs-products` или `make logs-product-management`. Показывает только логи подов, активных в момент вызова.

### Known issues
- Команду `make apply` скорее всего придется повторить два раза, так как файлы применяются в случайном порядке и может быть ошибка 
с созданием ресурса в несуществующем неймспейсе.
- На шагах 3-5 могут быть ошибки, если сервисы еще не успели подняться. Достаточно повторить команду чуть позже.
- Elasticsearch занимает много памяти, иногда для `make release` может не хватать памяти. В таком случае нужно заскейлить
elastic до 0 `kubectl scale statefulset elasticsearch -n products-infra --replicas=0`, а потом вернуть обратно с `--replicas=1`. 

## Примеры запросов

Создать новый продукт:
```
curl localhost/product -d '{"name":"test","price":100}' -H 'X-User-Id: 1'
```

Сделать поиск по продуктам (параметры опциональны):
```
curl localhost/products/search?name=test&from=10&to=100
```
from, to - ценовой диапазон.
