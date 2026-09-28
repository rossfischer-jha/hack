#! /bin/sh

set -x

ls -ld /elec-wire-*;

## TODO: move to bindings system and put in loop

echo "Migrating Wire Service"
migrate -database "spanner://projects/${PROJECT}/instances/${INSTANCE}/databases/elec-wire-service-db?x-clean-statements=true&x-dml-comment-flag=DML" -verbose -path /elec-wire-service-migrations up

echo "Migrating Wire Entity Service"
migrate -database "spanner://projects/${PROJECT}/instances/${INSTANCE}/databases/elec-wire-entity-service-db?x-clean-statements=true&x-dml-comment-flag=DML" -verbose -path /elec-wire-entity-service-migrations up

echo "Migrating Wire Approval Service"
migrate -database "spanner://projects/${PROJECT}/instances/${INSTANCE}/databases/elec-wire-approval-db?x-clean-statements=true&x-dml-comment-flag=DML" -verbose -path /elec-wire-approval-service-migrations up

echo "Migrating Wire Draft Service"
migrate -database "spanner://projects/${PROJECT}/instances/${INSTANCE}/databases/elec-wire-draft-service-db?x-clean-statements=true&x-dml-comment-flag=DML" -verbose -path /elec-wire-draft-service-migrations up

echo "Migrating Wire Money Transfer Service"
migrate -database "spanner://projects/${PROJECT}/instances/${INSTANCE}/databases/elec-money-transfer-service-db?x-clean-statements=true&x-dml-comment-flag=DML" -verbose -path /elec-money-transfer-service-migrations up

echo "Migrating Incoming Wire Processor Service"
migrate -database "spanner://projects/${PROJECT}/instances/${INSTANCE}/databases/elec-inc-wire-processor-db?x-clean-statements=true&x-dml-comment-flag=DML" -verbose -path /elec-incoming-wire-processor-migrations up

echo "Migrating Wire Fee Service"
migrate -database "spanner://projects/${PROJECT}/instances/${INSTANCE}/databases/elec-wire-fee-db?x-clean-statements=true&x-dml-comment-flag=DML" -verbose -path /elec-wire-fee-service-migrations up

echo "Migrating Wire Test Service"
migrate -database "spanner://projects/${PROJECT}/instances/${INSTANCE}/databases/elec-wire-test-db?x-clean-statements=true&x-dml-comment-flag=DML" -verbose -path /elec-wire-test-migrations up

