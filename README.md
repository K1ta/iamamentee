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

## Шардирование

Шардирование реализовано для таблицы products сервиса products по id. Шарды управляются в statefulset postgres через количество реплик.
Для добавления шардов нужно настроить параметры окружения products:
```
APP_DB_CONNECTIONS=postgres-0>[connection dsn]
APP_SHARDS=products-0:postgres-0
``` 
`APP_DB_CONNECTIONS` содержит мапу коннектов к постгресу. Key-value разделяются символом `>`, а пары - запятой. `APP_SHARDS` маппит название шарда
на коннект к этому шарду. 

Продукты в сервисе `product-management` шардированы по `userID`, в сервисе `products` - по `ID`.

### Миграции шарда

Изменение количества шардов делается в несколько этапов, и какое-то время сервис должен будет работать, зная предыдущую конфигурацию шардов. Для
этого используется переменная `APP_PREV_SHARDS` - она содержит предыдущее значение `APP_SHARDS`. Если оба параметра заданы, то сервис работает в режиме
миграции шардов. Для операций записи он использует новые шарды, для чтения - сначала новые, если не находит данные - то старые шарды.

Алгоритм для миграции
1. Запуск нового шарда, если это увеличение количества шардов
2. Запуск сервиса в режиме миграции - для этого переименовать `APP_SHARDS` в `APP_PREV_SHARDS`, затем задать новую конфигурацию шардов в `APP_SHARDS`
3. Миграция данных в новые шарды. Для этого нужно запустить джобу: `kubectl apply -f k8s-jobs/[service]/shards-migration.yaml`
4. После окончания миграции нужно запустить удаление данных со старых шардов, для этого запустить джобу: `kubectl apply -f k8s-jobs/[service]/shards-cleanup.yaml`
5. Удалить выполненные джобы
6. Убрать переменную `APP_PREV_SHARDS` из сервиса, теперь он рабоатет только с новыми шардами
7. Если это уменьшение количества шардов, то старые шарды можно удалить

#### Ошибки джоб при миграции

Джобы для миграции/удаления данных запускают горутины для каждого старого шарда. Эта горутина проходит по всем записям в products, пересчитывает score и
при необходимости пишет запись в новый шард или удаляет ее. Если горутина упадет с ошибкой, она выдаст лог с токеном `RESTART_DATA`. После окончания джобы
нужно проверить логи, и если какие-то горутины завершились с ошибкой, нужно обновить `MIGRATOR_PREV_SHARDS_START_FROM` в конфиге джобы и перезапустить ее.
Эта переменная содержит id продукта, с которого начинается скан. Пример заполнения:
```
MIGRATOR_PREV_SHARDS_START_FROM=postgres-0:123,postgres-1:10000
``` 

Если какие-то шарды успешно обработались, но нужно перезапустить джобу, их можно исключить из миграции:
```
MIGRATOR_EXCLUDED_PREV_SHARDS=postgres-0,postgres1
```

При первом запуске все эти переменные нужно оставить пустыми.

## Тестирование shared_buffers

Посмотреть параметры постгреса:
```sql
SELECT name, pg_size_pretty(setting::BIGINT * 
    CASE unit 
        WHEN '8kB' THEN 8192
        WHEN 'kB' THEN 1024
        ELSE 1
    END
) AS size
FROM pg_settings
WHERE name IN ('shared_buffers', 'work_mem', 'effective_cache_size', 'maintenance_work_mem');
```

Запрос для генерации записей в orders:
```sql
INSERT INTO orders (user_id, status, attempts, max_attempts, next_attempt_after)
SELECT
    (random() * 1000000)::BIGINT,
    'processing',
    0,
    -1,
    now()
FROM generate_series(1, 50000000);

INSERT INTO items (order_id, product_id, amount, price)
SELECT
    o.id,
    (random() * 100000)::BIGINT,
    (random() * 10)::INT + 1,
    (random() * 10000)::BIGINT + 100
FROM orders o;
```

Сброс page cache:
```shell
minikube ssh
sudo sh -c 'echo 3 > /proc/sys/vm/drop_caches'
```

Рестарт postgres пода:
```shell
kubectl rollout restart -n orders-infra statefulset/postgres
```

Очистка метрик перед стартом:
```sql
SELECT pg_stat_reset();
```

Просмотр, что лежит в кеше
```sql
CREATE EXTENSION pg_buffercache;

SELECT
    c.relname,
    count(*) AS buffers_in_cache,
    pg_size_pretty(count(*) * 8192) AS size_in_cache
FROM pg_buffercache b
JOIN pg_class c ON c.relfilenode = b.relfilenode
WHERE c.relname IN ('orders', 'items', 'items_order_id_idx', 'orders_pkey')
GROUP BY c.relname
ORDER BY buffers_in_cache DESC;
```

Запрос для снятия метрик:
```sql
SELECT
    relname,
    heap_blks_read,
    heap_blks_hit,
    round(
        heap_blks_hit::numeric / nullif(heap_blks_hit + heap_blks_read, 0) * 100,
        2
    ) AS heap_hit_pct,
    idx_blks_read,
    idx_blks_hit,
    round(
        idx_blks_hit::numeric / nullif(idx_blks_hit + idx_blks_read, 0) * 100,
        2
    ) AS idx_hit_pct
FROM pg_statio_user_tables
WHERE relname IN ('orders', 'items');
```

Запуск k6 скрипта:
```shell
K6_WEB_DASHBOARD=true \
K6_WEB_DASHBOARD_HOST=0.0.0.0 \
nohup k6 run /scripts/load_test.js > /tmp/k6.log 2>&1 & \
echo $!
```

### Итерация первая

Настройки постгрес:
```
         name         |  size
----------------------+---------
 effective_cache_size | 4096 MB
 maintenance_work_mem | 64 MB
 shared_buffers       | 128 MB
 work_mem             | 4096 kB
```

Значение перед стартом:
```
 relname | heap_blks_read | heap_blks_hit | cache_hit_pct
---------+----------------+---------------+---------------
 orders  |              0 |             0 |
 items   |              0 |             0 |
```

T=30s
```
 relname | heap_blks_read | heap_blks_hit | cache_hit_pct
---------+----------------+---------------+---------------
 orders  |         127423 |          1072 |          0.83
 items   |         127305 |          1190 |          0.93
```

T=1m30s
```
 relname | heap_blks_read | heap_blks_hit | cache_hit_pct
---------+----------------+---------------+---------------
 orders  |         396008 |          3325 |          0.83
 items   |         395610 |          3724 |          0.93
```

T=6m (добавил статистику по индексам)
```
 relname | heap_blks_read | heap_blks_hit | heap_hit_pct | idx_blks_read | idx_blks_hit | idx_hit_pct
---------+----------------+---------------+--------------+---------------+--------------+-------------
 orders  |        1693176 |         14193 |         0.83 |       1659998 |      5170156 |       75.70
 items   |        1691496 |         15875 |         0.93 |       1660025 |      5170807 |       75.70
 ```

 ### Итерация вторая. Добавил памяти в shared_buffers

 Параметры постгреса:
 ```
          name         |  size
----------------------+---------
 effective_cache_size | 9216 MB
 maintenance_work_mem | 64 MB
 shared_buffers       | 6144 MB
 work_mem             | 4096 kB
 ```

T=0:
```
 relname | heap_blks_read | heap_blks_hit | heap_hit_pct | idx_blks_read | idx_blks_hit | idx_hit_pct
---------+----------------+---------------+--------------+---------------+--------------+-------------
 orders  |              0 |             0 |              |             0 |            0 |
 items   |              0 |             0 |              |             0 |            0 |
```

T=30s
```
 relname | heap_blks_read | heap_blks_hit | heap_hit_pct | idx_blks_read | idx_blks_hit | idx_hit_pct
---------+----------------+---------------+--------------+---------------+--------------+-------------
 orders  |         115708 |         17358 |        13.04 |         85289 |       447104 |       83.98
 items   |         113794 |         19272 |        14.48 |         85465 |       447057 |       83.95
```

T=1m30s
```
 relname | heap_blks_read | heap_blks_hit | heap_hit_pct | idx_blks_read | idx_blks_hit | idx_hit_pct
---------+----------------+---------------+--------------+---------------+--------------+-------------
 orders  |         245722 |        103452 |        29.63 |        126512 |      1270544 |       90.94
 items   |         236382 |        112792 |        32.30 |        126378 |      1271038 |       90.96
```

T=2m30s
```
 relname | heap_blks_read | heap_blks_hit | heap_hit_pct | idx_blks_read | idx_blks_hit | idx_hit_pct
---------+----------------+---------------+--------------+---------------+--------------+-------------
 orders  |         250959 |        109286 |        30.34 |        127338 |      1314465 |       91.17
 items   |         241154 |        119091 |        33.06 |        127260 |      1315366 |       91.18
```

Итог - упал, стало не хватать памяти. RPS снизилось с ~5k до 150

### Итерация третья - уменьшил shared_buffers

 Параметры постгреса:
 ```
          name         |  size
----------------------+---------
 effective_cache_size | 7168 MB
 maintenance_work_mem | 64 MB
 shared_buffers       | 3072 MB
 work_mem             | 4096 kB
 ```

T=30s
```
 relname | heap_blks_read | heap_blks_hit | heap_hit_pct | idx_blks_read | idx_blks_hit | idx_hit_pct
---------+----------------+---------------+--------------+---------------+--------------+-------------
 orders  |         118050 |         17779 |        13.09 |         86571 |       456903 |       84.07
 items   |         116086 |         19743 |        14.54 |         86653 |       456979 |       84.06
```

T=1m30s
```
 relname | heap_blks_read | heap_blks_hit | heap_hit_pct | idx_blks_read | idx_blks_hit | idx_hit_pct
---------+----------------+---------------+--------------+---------------+--------------+-------------
 orders  |         326576 |         82865 |        20.24 |        188551 |      1449371 |       88.49
 items   |         317715 |         91726 |        22.40 |        188672 |      1449408 |       88.48
```

T=2m30s
```
 relname | heap_blks_read | heap_blks_hit | heap_hit_pct | idx_blks_read | idx_blks_hit | idx_hit_pct
---------+----------------+---------------+--------------+---------------+--------------+-------------
 orders  |         532827 |        147233 |        21.65 |        288515 |      2431903 |       89.39
 items   |         517137 |        162923 |        23.96 |        288589 |      2432007 |       89.39
```

T=7m
```
 relname | heap_blks_read | heap_blks_hit | heap_hit_pct | idx_blks_read | idx_blks_hit | idx_hit_pct
---------+----------------+---------------+--------------+---------------+--------------+-------------
 orders  |        1494063 |        446295 |        23.00 |        754429 |      7007449 |       90.28
 items   |        1446820 |        493539 |        25.44 |        754459 |      7007865 |       90.28
```

T=9m30s
```
 relname | heap_blks_read | heap_blks_hit | heap_hit_pct | idx_blks_read | idx_blks_hit | idx_hit_pct
---------+----------------+---------------+--------------+---------------+--------------+-------------
 orders  |        1983125 |        598641 |        23.19 |        991384 |      9336311 |       90.40
 items   |        1919706 |        662054 |        25.64 |        991824 |      9336470 |       90.40
```
