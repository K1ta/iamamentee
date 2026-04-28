import time

import requests

from config import DEFAULT_USER_ID, DELIVERY_URL, ORDERS_URL, PAYMENTS_URL, POLL_INTERVAL, POLL_TIMEOUT


def retry_until_ok(fn, timeout=POLL_TIMEOUT, interval=POLL_INTERVAL):
    deadline = time.time() + timeout
    last_exc = None
    while time.time() < deadline:
        try:
            fn()
            return
        except Exception as e:
            last_exc = e
        time.sleep(interval)
    raise TimeoutError(f"Timed out after {timeout}s. Last error: {last_exc}")


def wait_for_order_status(order_id, target_status, timeout=POLL_TIMEOUT, user_id=DEFAULT_USER_ID):
    terminal = {"failed", "canceled", "completed"}
    deadline = time.time() + timeout
    while time.time() < deadline:
        r = requests.get(
            f"{ORDERS_URL}/orders",
            params={"order_id": order_id},
            headers={"X-User-ID": user_id},
        )
        r.raise_for_status()
        status = r.json()["status"]
        if status == target_status:
            return
        if status in terminal:
            raise AssertionError(
                f"Order {order_id} reached terminal status {status!r}, expected {target_status!r}"
            )
        time.sleep(POLL_INTERVAL)
    raise TimeoutError(f"Order {order_id} did not reach {target_status!r} within {timeout}s")


def _mock_payment(order_id, endpoint):
    def _try():
        r = requests.post(f"{PAYMENTS_URL}/payments/mock/{endpoint}", json={"order_id": order_id})
        r.raise_for_status()

    retry_until_ok(_try)


def _mock_delivery(order_id, endpoint):
    def _try():
        r = requests.post(f"{DELIVERY_URL}/delivery/mock/{endpoint}", json={"order_id": order_id})
        r.raise_for_status()

    retry_until_ok(_try)


def confirm_payment(order_id):
    _mock_payment(order_id, "success")


def fail_payment(order_id):
    _mock_payment(order_id, "fail")


def confirm_delivery(order_id):
    _mock_delivery(order_id, "success")


def fail_delivery(order_id):
    _mock_delivery(order_id, "fail")
