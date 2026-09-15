#!/usr/bin/env bash
# One-time bootstrap for a Huginn GCP VM. Run this from Cloud Shell, not
# from the VM itself — it needs project-level IAM permissions the VM's
# own service account deliberately doesn't have.
#
# Grants Huginn's service account permission to manage firewall rules,
# and widens the VM's access scopes so it can actually use that grant.
# Both are one-time, per-VM prerequisites — Huginn's own code can't
# grant itself permissions it doesn't have. After this, firewall ports
# open automatically on every start and every config save.
set -euo pipefail

read -p "GCP project ID: " PROJECT_ID
read -p "VM instance name: " INSTANCE_NAME
read -p "VM zone (e.g. us-central1-a): " ZONE

SERVICE_ACCOUNT=$(gcloud compute instances describe "$INSTANCE_NAME" --zone="$ZONE" \
  --format='value(serviceAccounts[0].email)')

echo "Granting roles/compute.securityAdmin to $SERVICE_ACCOUNT..."
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SERVICE_ACCOUNT" \
  --role="roles/compute.securityAdmin"

echo "Stopping $INSTANCE_NAME to widen its access scopes..."
gcloud compute instances stop "$INSTANCE_NAME" --zone="$ZONE"

gcloud compute instances set-service-account "$INSTANCE_NAME" --zone="$ZONE" \
  --service-account="$SERVICE_ACCOUNT" \
  --scopes=cloud-platform

echo "Restarting $INSTANCE_NAME..."
gcloud compute instances start "$INSTANCE_NAME" --zone="$ZONE"

echo "Done. Huginn can now manage its own firewall rules."
