#!/usr/bin/env bash
set -euo pipefail

PROJECT_NAME="ce-vpc-power-schedule"
IBM_APIKEY=${IBM_APIKEY:-}
IBM_REGION=${IBM_REGION:-us-south}

if [[ -z "$IBM_APIKEY" ]]; then echo "ERROR: IBM_APIKEY is required."; exit 1; fi
if [[ -z "$IBM_REGION" ]]; then echo "ERROR: IBM_REGION is required."; exit 1; fi

ibmcloud login --apikey "$IBM_APIKEY" -r "$IBM_REGION"

if ibmcloud ce project get --name "$PROJECT_NAME" >/dev/null 2>&1; then
  echo "[+] Deleting Code Engine project: $PROJECT_NAME"
  ibmcloud ce project delete --name "$PROJECT_NAME" --force --hard
  echo "[+] Project deleted successfully"
else
  echo "[!] Project $PROJECT_NAME not found"
fi

echo "[+] Cleanup complete"
