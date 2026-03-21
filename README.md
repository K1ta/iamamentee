# Учебный проект

## Запуск

1. Запустить кластер миникуба `make minikube-up`
2. Настроить k8s `make apply` - скорее всего придется два раза повторить, так как файлы применяются в случайном порядке и могут
быть ошибки с неймспейсами
3. Установить [goose](https://github.com/pressly/goose) для миграций: `brew install goose`
4. Создать топик в кафке `make create-topic-product-management.product`
5. Задеплоить сервисы `make release`
6. Открыть порт в ингрес `make port-forward-ingress`

После этого можно отправлять запросы по `localhost:8080`:
```
curl localhost:8080/product -d '{"name":"test","price":100}' -H 'X-User-Id: 1'
curl localhost:8080/products/search
```

На всех этапах могут быть ошибки, если сервисы еще не успели подняться. Например, топик может не создасться, потому что кафка еще не готова. Достаточно повторить
команду чуть позже.