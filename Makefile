.SILENT:

SERVICES := product-management products orders payments delivery

LOCAL_PORT=9999
DB_PORT=5432
PG_PASSWORD=password
PG_USER=admin
ES_PORT=9200

# backend services
build-%:
	eval $$(minikube docker-env) && \
	docker build -t $*:latest backend/$*

deploy-%:
	if [ -f backend/$*/dev.env ]; then \
		kubectl create configmap $*-config \
		--from-env-file=backend/$*/dev.env \
		-n $* \
		--dry-run=client -o yaml | kubectl apply -f -; \
	fi
	kubectl rollout restart deployment $* -n $*

.PHONY: migrate-up-%
migrate-up-%:
	make migrate-main-up-$*
	make migrate-shards-up-$*

migrate-shards-up-%:
	if [ -d backend/$*/migrations/sharded ] && [ -n "$$(ls -A backend/$*/migrations/sharded)" ]; then \
        echo "Running sharded migrations..."; \
		pods="$(shell kubectl get pods -n $*-infra -l app=postgres -o jsonpath='{.items[*].metadata.name}')"; \
		echo $$pods; \
		for pod in $$pods; do \
			echo "Migrating up $$pod"; \
			kubectl port-forward $$pod -n $*-infra $(LOCAL_PORT):$(DB_PORT) & PF_PID=$$!; \
			echo PID=$$PF_PID; \
			sleep 1; \
			goose postgres "postgres://$(PG_USER):$(PG_PASSWORD)@localhost:$(LOCAL_PORT)/$*?sslmode=disable" up -dir backend/$*/migrations/sharded; \
			kill $$PF_PID; \
		done; \
    fi

migrate-main-up-%:
	if [ -d backend/$*/migrations/main ] && [ -n "$$(ls -A backend/$*/migrations/main)" ]; then \
        echo "Running main migrations..."; \
		kubectl port-forward postgres-0 -n $*-infra $(LOCAL_PORT):$(DB_PORT) & PF_PID=$$!; \
		echo PID=$$PF_PID; \
		sleep 1; \
		goose postgres "postgres://$(PG_USER):$(PG_PASSWORD)@localhost:$(LOCAL_PORT)/$*?sslmode=disable" up -dir backend/$*/migrations/main; \
		kill $$PF_PID; \
    fi

migrate-down-%:
	make migrate-main-down-$*
	make migrate-shards-down-$*

migrate-shards-down-%:
	if [ -d backend/$*/migrations/sharded ] && [ -n "$$(ls -A backend/$*/migrations/sharded)" ]; then \
        echo "Running sharded migrations..."; \
		pods="$(shell kubectl get pods -n $*-infra -l app=postgres -o jsonpath='{.items[*].metadata.name}')"; \
		echo $$pods; \
		for pod in $$pods; do \
			echo "Migrating down $$pod"; \
			kubectl port-forward $$pod -n $*-infra $(LOCAL_PORT):$(DB_PORT) & PF_PID=$$!; \
			echo PID=$$PF_PID; \
			sleep 1; \
			goose postgres "postgres://$(PG_USER):$(PG_PASSWORD)@localhost:$(LOCAL_PORT)/$*?sslmode=disable" down -dir backend/$*/migrations/sharded; \
			kill $$PF_PID; \
		done; \
    fi

migrate-main-down-%:
	if [ -d backend/$*/migrations/main ] && [ -n "$$(ls -A backend/$*/migrations/main)" ]; then \
        echo "Running main migrations..."; \
		kubectl port-forward postgres-0 -n $*-infra $(LOCAL_PORT):$(DB_PORT) & PF_PID=$$!; \
		echo PID=$$PF_PID; \
		sleep 1; \
		goose postgres "postgres://$(PG_USER):$(PG_PASSWORD)@localhost:$(LOCAL_PORT)/$*?sslmode=disable" down -dir backend/$*/migrations/main; \
		kill $$PF_PID; \
    fi

.PHONY: release release-%
release-%:
	make migrate-up-$*
	make build-$*
	make deploy-$*

release: $(SERVICES:%=release-%)

create-mappings-%:
	for f in backend/$*/mappings/*.json; do \
		echo "Creating mapping $$f?ignore=400"; \
		cat $$f | kubectl exec elasticsearch-0 -n $*-infra -i -- \
			curl -X PUT -s "http://localhost:9200/$$(basename $$f .json)" \
			-H 'Content-Type: application/json' \
			-d @-; \
	done

# k8s
apply:
	kubectl apply -f ./k8s -R --wait
	kubectl label namespace products istio-injection=enabled
	kubectl label namespace product-management istio-injection=enabled

port-forward-%:
	kubectl port-forward svc/$* 8080:80 -n $*

logs-%:
	kubectl logs -f -l app=$* -n $*

minikube-up:
	minikube start --driver=docker --memory=12288 --cpus=4 --disk-size=20gb
	minikube addons enable metrics-server
	istioctl install --set profile=demo -y

minikube-down:
	minikube stop
	minikube delete

minikube-restart: minikube-down minikube-up

restart-kafka:
	kubectl rollout restart statefulset kafka -n kafka-common

create-topic-%:
	kubectl exec -it kafka-0 -n kafka-common -- \
	/opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 \
	--create --topic $* \
	--partitions 3 \
	--replication-factor 2