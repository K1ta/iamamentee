## Отмена саги при ошибке оплаты
**Что изменить:**
1. payments: переводить платеж для заказа по статусам failing->failed, отправлять cancel-запрос в product-management
2. product-management: обрабатывать cancel-запроса, переводить резервацию  done->compensating->compensated->canceled, отправлять cancel-запрос в orders
**Готово когда:** e2e тест test_payment_failure_cancels_order проходит

## Отмена саги при ошибке доставки
**Что изменить:** 
1. delivery: переводить доставку по статусам failing->failed, отправлять cancel-запрос в payments
2. payments: обрабатывать cancel-запрос, переводить платеж по статусам done->compensating->compensated->canceled, отправлять cancel-запрос в product-management
**Готово когда:** e2e тест test_delivery_failure_cancels_order проходит