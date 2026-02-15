#!/usr/bin/env bash
set -euo pipefail

# Emits JSON consumed by kfin MCP pricing provider:
# {"cpu_per_hour":<num>,"mem_per_gb_hour":<num>}
#
# Usage:
#   ./scripts/aws-pricing-rates.sh [instance-type] [location]
#   ./scripts/aws-pricing-rates.sh --mode explicit-rates --cpu-rate 0.031 --mem-rate 0.0045
#
# Example:
#   ./scripts/aws-pricing-rates.sh c6a.large "US East (Ohio)"
#   ./scripts/aws-pricing-rates.sh --mode explicit-rates --cpu-rate 0.031 --mem-rate 0.0045
#
# Optional env vars:
#   AWS_PROFILE                (default: default)
#   AWS_PRICING_API_REGION     (default: us-east-1)
#   AWS_EC2_INSTANCE_TYPE      (default: c6a.large)
#   AWS_EC2_LOCATION           (default: US East (Ohio))
#   AWS_PRICING_MODE           (default: split-instance)
#   AWS_CPU_PER_HOUR           (used in explicit-rates mode)
#   AWS_MEM_PER_GB_HOUR        (used in explicit-rates mode)

if ! command -v aws >/dev/null 2>&1; then
  echo "ERROR: aws CLI is required" >&2
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "ERROR: jq is required" >&2
  exit 1
fi

instance_type="${1:-${AWS_EC2_INSTANCE_TYPE:-c6a.large}}"
location="${2:-${AWS_EC2_LOCATION:-US East (Ohio)}}"
aws_profile="${AWS_PROFILE:-default}"
pricing_api_region="${AWS_PRICING_API_REGION:-us-east-1}"
mode="${AWS_PRICING_MODE:-split-instance}"
cpu_rate="${AWS_CPU_PER_HOUR:-}"
mem_rate="${AWS_MEM_PER_GB_HOUR:-}"
positional_idx=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)
      mode="$2"
      shift 2
      ;;
    --cpu-rate)
      cpu_rate="$2"
      shift 2
      ;;
    --mem-rate)
      mem_rate="$2"
      shift 2
      ;;
    --instance-type)
      instance_type="$2"
      shift 2
      ;;
    --location)
      location="$2"
      shift 2
      ;;
    *)
      # Backward-compatible positional arguments: instance_type location
      if (( positional_idx == 0 )); then
        instance_type="$1"
      elif (( positional_idx == 1 )); then
        location="$1"
      else
        echo "ERROR: unknown argument: $1" >&2
        exit 1
      fi
      positional_idx=$((positional_idx + 1))
      shift
      ;;
  esac
done

if [[ "${mode}" == "explicit-rates" ]]; then
  if [[ -z "${cpu_rate}" || -z "${mem_rate}" ]]; then
    echo "ERROR: explicit-rates mode requires --cpu-rate and --mem-rate (or AWS_CPU_PER_HOUR/AWS_MEM_PER_GB_HOUR)" >&2
    exit 1
  fi
  jq -n \
    --argjson cpu "${cpu_rate}" \
    --argjson mem "${mem_rate}" \
    '{cpu_per_hour:$cpu, mem_per_gb_hour:$mem}'
  exit 0
fi

if [[ "${mode}" != "split-instance" ]]; then
  echo "ERROR: invalid mode '${mode}' (expected: split-instance or explicit-rates)" >&2
  exit 1
fi

raw_json="$(
  aws --profile "${aws_profile}" --region "${pricing_api_region}" pricing get-products \
    --service-code AmazonEC2 \
    --filters \
      Type=TERM_MATCH,Field=instanceType,Value="${instance_type}" \
      Type=TERM_MATCH,Field=location,Value="${location}" \
      Type=TERM_MATCH,Field=operatingSystem,Value=Linux \
      Type=TERM_MATCH,Field=preInstalledSw,Value=NA \
      Type=TERM_MATCH,Field=tenancy,Value=Shared \
      Type=TERM_MATCH,Field=capacitystatus,Value=Used \
    --max-results 1 \
    --query 'PriceList[0]' \
    --output text
)"

if [[ -z "${raw_json}" || "${raw_json}" == "None" ]]; then
  echo "ERROR: no pricing result for instance_type=${instance_type}, location=${location}" >&2
  exit 1
fi

hourly_usd="$(jq -r '
  .terms.OnDemand
  | to_entries[0].value.priceDimensions
  | to_entries[0].value.pricePerUnit.USD
' <<<"${raw_json}")"
vcpu="$(jq -r '.product.attributes.vcpu' <<<"${raw_json}")"
memory_raw="$(jq -r '.product.attributes.memory' <<<"${raw_json}")"

if [[ -z "${hourly_usd}" || -z "${vcpu}" || -z "${memory_raw}" || "${hourly_usd}" == "null" || "${vcpu}" == "null" || "${memory_raw}" == "null" ]]; then
  echo "ERROR: failed to parse pricing payload fields" >&2
  exit 1
fi

memory_gib="$(sed -E 's/,//g; s/[[:space:]]*GiB$//' <<<"${memory_raw}")"

cpu_per_hour="$(awk -v p="${hourly_usd}" -v c="${vcpu}" 'BEGIN { if (c<=0) exit 1; printf "%.8f", p/c }')" || {
  echo "ERROR: invalid vcpu value: ${vcpu}" >&2
  exit 1
}
mem_per_gb_hour="$(awk -v p="${hourly_usd}" -v m="${memory_gib}" 'BEGIN { if (m<=0) exit 1; printf "%.8f", p/m }')" || {
  echo "ERROR: invalid memory GiB value: ${memory_gib} (raw=${memory_raw})" >&2
  exit 1
}

jq -n \
  --argjson cpu "${cpu_per_hour}" \
  --argjson mem "${mem_per_gb_hour}" \
  '{cpu_per_hour:$cpu, mem_per_gb_hour:$mem}'
