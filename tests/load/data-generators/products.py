import psycopg2
import random
import time
import string

HOST = "localhost"
PORT = 15432
DB_NAME = "products"
USER="admin"
PASSWORD="password"

DATA_COUNT=10000

def random_name(length=10):
    return ''.join(random.choices(string.ascii_lowercase, k=length))

def random_price():
    return round(random.uniform(1, 1000), 2)

def random_user_id():
    return random.randint(1, 1000)

def main():
    conn = psycopg2.connect(
        host=HOST,
        port=PORT,
        dbname=DB_NAME,
        user=USER,
        password=PASSWORD
    )
    conn.autocommit = True
    cur = conn.cursor()

    for i in range(DATA_COUNT):
        user_id = random_user_id()
        name = random_name()
        price = random_price()

        cur.execute(
            "INSERT INTO products (user_id, name, price) VALUES (%s, %s, %s);",
            (user_id, name, price)
        )

        if i % 100 == 0 and i != 0:
            print(f"Inserted {i} rows...")

    end = time.time()

    cur.close()
    conn.close()


if __name__ == "__main__":
    main()