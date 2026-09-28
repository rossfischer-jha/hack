up:
	docker compose up -d

down:
	docker compose down --remove-orphans

restart:
	docker compose down --remove-orphans
	docker compose up -d

restart-pubsub:
	docker compose down --remove-orphans pubsub pubsub-init
	docker compose up -d pubsub pubsub-init

restart-spanner:
	docker compose down --remove-orphans spanner spanner-init spanner-migrate
	docker compose up -d spanner spanner-init spanner-migrate
