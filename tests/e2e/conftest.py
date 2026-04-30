import pytest
import requests

from config import DEFAULT_USER_ID, ORDERS_URL, PRODUCT_MANAGEMENT_URL


@pytest.fixture
def create_product():
    def _create(name="Test Product", price=1000, user_id=DEFAULT_USER_ID):
        r = requests.post(
            f"{PRODUCT_MANAGEMENT_URL}/product/",
            json={"name": name, "price": price},
            headers={"X-User-ID": user_id},
        )
        r.raise_for_status()
        print(f"product created: {r.json()['id']}")
        return r.json()

    return _create


@pytest.fixture
def create_order():
    def _create(product_id, amount=1, user_id=DEFAULT_USER_ID):
        r = requests.post(
            f"{ORDERS_URL}/orders/create",
            json={"items": [{"product_id": product_id, "amount": amount}]},
            headers={"X-User-ID": user_id},
        )
        r.raise_for_status()
        print(f"order created: {r.json()['id']}")
        return r.json()

    return _create
