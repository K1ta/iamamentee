.SILENT:

SERVICES := product-management products orders

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

migrate-up-%:
	pods="$(shell kubectl get pods -n $*-infra -l app=postgres -o jsonpath='{.items[*].metadata.name}')"; \
	echo $$pods; \
	for pod in $$pods; do \
		echo "Migrating up $$pod"; \
		kubectl port-forward $$pod -n $*-infra $(LOCAL_PORT):$(DB_PORT) & PF_PID=$$!; \
		echo PID=$$PF_PID; \
		sleep 1; \
		goose postgres "postgres://$(PG_USER):$(PG_PASSWORD)@localhost:$(LOCAL_PORT)/$*?sslmode=disable" up -dir backend/$*/migrations; \
		kill $$PF_PID; \
	done

migrate-down-%:
	pods="$(shell kubectl get pods -n $*-infra -l app=postgres -o jsonpath='{.items[*].metadata.name}')"; \
	echo $$pods; \
	for pod in $$pods; do \
		echo "Migrating down $$pod"; \
		kubectl port-forward $$pod -n $*-infra $(LOCAL_PORT):$(DB_PORT) & PF_PID=$$!; \
		echo PID=$$PF_PID; \
		sleep 1; \
		goose postgres "postgres://$(PG_USER):$(PG_PASSWORD)@localhost:$(LOCAL_PORT)/$*?sslmode=disable" down -dir backend/$*/migrations; \
		kill $$PF_PID; \
	done

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