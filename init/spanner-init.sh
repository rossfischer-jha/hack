#! /bin/bash
set -eu

DATABASES="elec-wire-service-db,elec-wire-entity-service-db,elec-wire-approval-service-db,elec-wire-draft-service-db,elec-wire-test-db,elec-money-transfer-service-db"

gcloud config configurations create emulator
gcloud config set auth/disable_credentials true
gcloud config set project ${PROJECT}
gcloud config set api_endpoint_overrides/spanner "http://local-spanner-db:9020/"
gcloud config set auth/disable_credentials true
gcloud spanner instances create ${INSTANCE} --config=emulator-config --description=Emulator --nodes=1


# Set comma as delimiter and read into an array
IFS=',' read -ra dbs <<< "$DATABASES"

# Now iterate
for db in "${dbs[@]}"; do
    echo "Creating database: ${db} for instance: ${INSTANCE}"
    gcloud spanner databases create "${db}" --instance="${INSTANCE}"
done
