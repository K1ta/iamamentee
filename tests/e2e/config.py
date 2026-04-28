import os

ORDERS_URL = os.getenv("ORDERS_URL", "http://localhost")
PRODUCT_MANAGEMENT_URL = os.getenv("PRODUCT_MANAGEMENT_URL", "http://localhost")
PAYMENTS_URL = os.getenv("PAYMENTS_URL", "http://localhost")
DELIVERY_URL = os.getenv("DELIVERY_URL", "http://localhost")

DEFAULT_USER_ID = "1"
POLL_INTERVAL = 0.5
POLL_TIMEOUT = 30
