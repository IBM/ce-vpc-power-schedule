#!/usr/bin/env bash
set -euo pipefail

PROJECT_NAME="ce-vpc-power-schedule"
RESOURCE_GROUP=${RESOURCE_GROUP:-}
REGISTRY_IMAGE="private.icr.io/$PROJECT_NAME/$PROJECT_NAME:latest"
JOB_POWER_ON_NAME=${JOB_NAME:-ce-vpc-poweron-schedule--job}
JOB_POWER_OFF_NAME=${JOB_NAME:-ce-vpc-poweroff-schedule--job}
CONFIGMAP_FILE_NAME=${CONFIGMAP_NAME:-entities-and-exclusions--cm}
CONFIGMAP_NAME=${CONFIGMAP_NAME:-config--cm}
SECRET_NAME=${SECRET_NAME:-credentials}
IBM_APIKEY=${IBM_APIKEY:-}
IBM_REGION=${IBM_REGION:-us-south}

if [[ -z "$IBM_APIKEY" ]]; then echo "ERROR: IBM_APIKEY is required."; exit 1; fi
if [[ -z "$IBM_REGION" ]]; then echo "ERROR: IBM_REGION is required."; exit 1; fi
if [[ -z "$RESOURCE_GROUP" ]]; then echo "ERROR: RESOURCE_GROUP is required."; exit 1; fi

if [[ ! -f "schedule_template.yaml" ]]; then echo "ERROR: schedule_template.yaml not found."; exit 1; fi

ibmcloud login --apikey "$IBM_APIKEY" -r "$IBM_REGION"

if ! ibmcloud resource group "$RESOURCE_GROUP" >/dev/null 2>&1; then
  ibmcloud resource group-create "$RESOURCE_GROUP"
fi

ibmcloud target -g "$RESOURCE_GROUP"

if ibmcloud ce project get --name "$PROJECT_NAME" >/dev/null 2>&1; then
  ibmcloud ce project select --name "$PROJECT_NAME"
else
  ibmcloud ce project create --name "$PROJECT_NAME"
  ibmcloud ce project select --name "$PROJECT_NAME"
fi

echo "[+] Project: $PROJECT_NAME selected"

if ibmcloud ce secret get --name "$SECRET_NAME" >/dev/null 2>&1; then
  ibmcloud ce secret update --name "$SECRET_NAME" --from-literal IBM_APIKEY="$IBM_APIKEY"
else
  ibmcloud ce secret create --name "$SECRET_NAME" --from-literal IBM_APIKEY="$IBM_APIKEY"
fi

echo "[+] Secret ready: $SECRET_NAME"

if ibmcloud ce configmap get --name "$CONFIGMAP_FILE_NAME" >/dev/null 2>&1; then
  ibmcloud ce configmap update --name "$CONFIGMAP_FILE_NAME" --from-file schedule.yaml=schedule_template.yaml
else
  ibmcloud ce configmap create --name "$CONFIGMAP_FILE_NAME" --from-file schedule.yaml=schedule_template.yaml
fi

echo "[+] ConfigMap ready: $CONFIGMAP_FILE_NAME"

if ibmcloud ce configmap get --name "$CONFIGMAP_NAME" >/dev/null 2>&1; then
  ibmcloud ce configmap update \
    --name "$CONFIGMAP_NAME" \
    --from-literal IBM_REGION="$IBM_REGION" \
    --from-literal LOG_LEVEL="Info"
else
  ibmcloud ce configmap create \
    --name "$CONFIGMAP_NAME" \
    --from-literal IBM_REGION="$IBM_REGION" \
    --from-literal LOG_LEVEL="Info"
fi

echo "[+] ConfigMap ready: $CONFIGMAP_NAME"

ibmcloud ce buildrun delete -f --inf --name "${PROJECT_NAME}--build" || true
ibmcloud ce buildrun submit --name "${PROJECT_NAME}--build" --image "$REGISTRY_IMAGE" --source . --strategy dockerfile --size small --wait

ibmcloud ce job delete -f --inf --name "$JOB_POWER_ON_NAME" || true
ibmcloud ce job create \
  --name "$JOB_POWER_ON_NAME" \
  --mode "task" \
  --image "$REGISTRY_IMAGE" \
  --registry-secret "ce-auto-icr-private-global" \
  --env-from-secret "$SECRET_NAME" \
  --mount-configmap /app/config="$CONFIGMAP_FILE_NAME" \
  --env-from-configmap "$CONFIGMAP_NAME" \
  --cpu 0.25 --memory "0.5G" \
  --argument "powerOn"

echo "[+] Job ready: $JOB_POWER_ON_NAME"

ibmcloud ce job delete -f --inf --name "$JOB_POWER_OFF_NAME" || true
ibmcloud ce job create \
  --name "$JOB_POWER_OFF_NAME" \
  --mode "task" \
  --image "$REGISTRY_IMAGE" \
  --registry-secret "ce-auto-icr-private-global" \
  --env-from-secret "$SECRET_NAME" \
  --mount-configmap /app/config="$CONFIGMAP_FILE_NAME" \
  --env-from-configmap "$CONFIGMAP_NAME" \
  --cpu 0.25 --memory "0.5G" \
  --argument "powerOff"

echo "[+] Job ready: $JOB_POWER_OFF_NAME"

ibmcloud ce subscription cron delete -f --inf --name "poweron-working-days--cron" || true
ibmcloud ce subscription cron create \
  --name "poweron-working-days--cron" \
  --destination-type job \
  --destination "$JOB_POWER_ON_NAME" \
  --schedule '0 7 * * 1-5'

echo "[+] Cron subscription created: poweron-working-days--cron"

ibmcloud ce subscription cron delete -f --inf --name "poweroff-working-days--cron" || true
ibmcloud ce subscription cron create \
  --name "poweroff-working-days--cron" \
  --destination-type job \
  --destination "$JOB_POWER_OFF_NAME" \
  --schedule '0 19 * * 1-5'

echo "[+] Cron subscription created: poweroff-working-days--cron"

echo "[+] Done."
