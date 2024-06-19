#!/bin/bash
#
# Combine bid CSVs (from bidcollect) into a single CSV, and upload to R2/S3
#
set -e

# require directory as first argument
if [ -z "$1" ]; then
  echo "Usage: $0 <directory>"
  exit 1
fi

cd $1
date=$(basename $1)
ym=${date:0:7}
echo $date
echo ""

# ALL BIDS
fn_out="${date}_all.csv"
fn_out_zip="${fn_out}.zip"
rm -f $fn_out $fn_out_zip

echo "Combining all bids..."
first="1"
for fn in $(\ls all*); do
    echo "- ${fn}"
    if [ $first == "1" ]; then
        head -n 1 $fn > $fn_out
        first="0"
    fi
    tail -n +2 $fn >> $fn_out
done

echo "Lines (all bids):"
wc -l $fn_out

echo "Source types (all bids):"
clickhouse local -q "SELECT source_type, COUNT(source_type) FROM '$fn_out' GROUP BY source_type ORDER BY source_type;"

zip ${fn_out_zip} $fn_out
echo "Wrote ${fn_out_zip}"
rm -f $fn_out
rm -f all*.csv

# Upload
if [[ "${UPLOAD}" != "0" ]]; then
    echo "Uploading to R2 and S3..."
    aws --profile r2 s3 cp --no-progress "${fn_out_zip}" "s3://relayscan-bidarchive/ethereum/mainnet/${ym}/" --endpoint-url "https://${CLOUDFLARE_R2_ACCOUNT_ID}.r2.cloudflarestorage.com"
    aws --profile s3 s3 cp --no-progress "${fn_out_zip}" "s3://relayscan-bidarchive/ethereum/mainnet/${ym}/"
fi

if [[ "${DEL}" == "1" ]]; then
    rm -f all*
fi

echo ""

# TOP BIDS
echo "Combining top bids..."
fn_out="${date}_top.csv"
fn_out_zip="${fn_out}.zip"
rm -f $fn_out $fn_out_zip

first="1"
for fn in $(\ls top*); do
    echo "- ${fn}"
    if [ $first == "1" ]; then
        head -n 1 $fn > $fn_out
        first="0"
    fi
    tail -n +2 $fn >> $fn_out
done

echo "Lines (top bids):"
wc -l $fn_out

echo "Source types (top bids):"
clickhouse local -q "SELECT source_type, COUNT(source_type) FROM '$fn_out' GROUP BY source_type ORDER BY source_type;"

zip ${fn_out_zip} $fn_out
echo "Wrote ${fn_out_zip}"
rm -f $fn_out
rm -f top*.csv

# Upload
if [[ "${UPLOAD}" != "0" ]]; then
    echo "Uploading to R2 and S3..."
    aws --profile r2 s3 cp --no-progress "${fn_out_zip}" "s3://relayscan-bidarchive/ethereum/mainnet/${ym}/" --endpoint-url "https://${CLOUDFLARE_R2_ACCOUNT_ID}.r2.cloudflarestorage.com"
    aws --profile s3 s3 cp --no-progress "${fn_out_zip}" "s3://relayscan-bidarchive/ethereum/mainnet/${ym}/"
fi

if [[ "${DEL}" == "1" ]]; then
    rm -f top*
fi
