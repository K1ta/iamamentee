from locust import HttpUser, task, between

class ProductUser(HttpUser):
    wait_time = between(1, 3)

    # @task(3)
    # def view_products(self):
    #     self.client.get("/products")

    # @task(1)
    # def create_product(self):
    #     self.client.post("/product", json={"name":"productX"})

    @task(1)
    def search_products(self):
        self.client.get("http://localhost/products/search")