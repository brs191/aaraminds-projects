#!/usr/bin/env bash
# Phase 0 - Workstream C: provision a DEV Azure PostgreSQL Flexible Server (PG16) + Apache AGE.
# Throwaway, for the traversal benchmark only. Run teardown at the bottom when done.
# Requires: az CLI logged in (az login), and a strong ADMIN_PASS.
set -euo pipefail

RG="${RG:-rg-repo-intel-phase0}"
LOC="${LOC:-eastus}"
SRV="${SRV:-pg-repo-intel-phase0-$RANDOM}"   # server name must be globally unique
ADMIN_USER="${ADMIN_USER:-pgadmin}"
ADMIN_PASS="${ADMIN_PASS:?set ADMIN_PASS to a strong password before running}"
SKU="${SKU:-Standard_D2ds_v5}"               # 2 vCPU general purpose; bump for a bigger graph
DB="${DB:-repointel}"
MY_IP="${MY_IP:?set MY_IP to your public IP (curl -s ifconfig.me) so you can connect}"

az group create -n "$RG" -l "$LOC" 1>/dev/null

az postgres flexible-server create \
  --resource-group "$RG" --name "$SRV" --location "$LOC" \
  --version 16 --tier GeneralPurpose --sku-name "$SKU" --storage-size 32 \
  --admin-user "$ADMIN_USER" --admin-password "$ADMIN_PASS" \
  --public-access None --yes 1>/dev/null

# Dev-only firewall rule for your machine. Lock this down for anything beyond the spike.
az postgres flexible-server firewall-rule create \
  -g "$RG" -n "$SRV" --rule-name devbox --start-ip-address "$MY_IP" --end-ip-address "$MY_IP" 1>/dev/null

# AGE needs BOTH params. shared_preload_libraries change forces a restart, so set it last.
az postgres flexible-server parameter set -g "$RG" -s "$SRV" --name azure.extensions --value AGE 1>/dev/null
az postgres flexible-server parameter set -g "$RG" -s "$SRV" --name shared_preload_libraries --value AGE 1>/dev/null
az postgres flexible-server restart -g "$RG" -s "$SRV" 1>/dev/null

az postgres flexible-server db create -g "$RG" -s "$SRV" -d "$DB" 1>/dev/null

HOST="$SRV.postgres.database.azure.com"
echo "Provisioned $HOST / db=$DB"
echo
echo "Enable AGE + load the graph:"
echo "  psql \"host=$HOST port=5432 dbname=$DB user=$ADMIN_USER sslmode=require\" -f setup.sql"
echo
echo "Then benchmark:"
echo "  export PGCONN='host=$HOST port=5432 dbname=$DB user=$ADMIN_USER password=*** sslmode=require'"
echo "  python benchmark.py --generate --nodes 2500 --avg-degree 4"
echo "  python benchmark.py --iterations 50"
echo
echo "Teardown when done:  az group delete -n $RG --yes --no-wait"
