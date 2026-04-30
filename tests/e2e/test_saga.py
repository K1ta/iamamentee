import pytest
import requests

from config import DEFAULT_USER_ID, ORDERS_URL
from helpers import (
    confirm_delivery,
    confirm_payment,
    fail_delivery,
    fail_payment,
    wait_for_order_status,
)


class TestOrderSagaHappyPath:
    def test_order_completes_after_payment_and_delivery(self, create_product, create_order):
        """
        Scenario: happy path
          Given a product exists
          When an order is created for that product
          And the reservation is confirmed (order reaches "processing")
          And payment is confirmed via mock
          And delivery is confirmed via mock
          Then the order reaches status "completed"
        """
        product = create_product(name="Widget", price=500)
        order = create_order(product_id=product["id"], amount=1)
        order_id = order["id"]

        wait_for_order_status(order_id, "processing")
        confirm_payment(order_id)
        confirm_delivery(order_id)
        wait_for_order_status(order_id, "completed")


class TestOrderCreation:
    def test_nonexistent_product_returns_error(self):
        """
        Scenario: order creation fails when product doesn't exist
          Given a product with id=9999999 does not exist
          When an order is created for that product
          Then the request returns an error
        """
        r = requests.post(
            f"{ORDERS_URL}/orders/create",
            json={"items": [{"product_id": 9999999, "amount": 1}]},
            headers={"X-User-ID": DEFAULT_USER_ID},
        )
        assert r.status_code == 500


class TestOrderSagaNegativePaths:
    def test_payment_failure_cancels_order(self, create_product, create_order):
        """
        Scenario: payment fails
          Given a product exists
          When an order is created
          And the reservation is confirmed (order reaches "processing")
          And payment fails via mock
          Then the order reaches status "canceled"
        """
        product = create_product(name="Widget", price=500)
        order = create_order(product_id=product["id"], amount=1)
        order_id = order["id"]

        wait_for_order_status(order_id, "processing")
        fail_payment(order_id)
        wait_for_order_status(order_id, "canceled")

    @pytest.mark.skip(reason="compensation for delivery failure not implemented")
    def test_delivery_failure_cancels_order(self, create_product, create_order):
        """
        Scenario: delivery fails after successful payment
          Given a product exists
          When an order is created
          And payment is confirmed
          And delivery fails via mock
          Then the order reaches status "canceled"
        """
        product = create_product(name="Widget", price=500)
        order = create_order(product_id=product["id"], amount=1)
        order_id = order["id"]

        wait_for_order_status(order_id, "processing")
        confirm_payment(order_id)
        fail_delivery(order_id)
        wait_for_order_status(order_id, "canceled")
