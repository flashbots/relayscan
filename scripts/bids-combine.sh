#!/bin/bash
#
# Combine bid CSVs (from bidcollect) into a single CSV
#
set -e

# require directory as first argument
if [ -z "$1" ]; then
  echo "Usage: $0 <directory>"
  exit 1
fi

cd $1
date=$(basename $1)
echo $date
echo ""

# ALL BIDS
fn_out="all_${date}.csv"
fn_out_zip="${fn_out}.zip"
fn_out_gz="${fn_out}.gz"
rm -f $fn_out $fn_out_zip $fn_out_gz

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
zip ${fn_out_zip} $fn_out
echo "Wrote ${fn_out_zip}"
gzip $fn_out
echo "Wrote ${fn_out_gz}"
rm -f $fn_out

echo ""

# TOP BIDS
echo "Combining top bids..."
fn_out="top_${date}.csv"
fn_out_zip="${fn_out}.zip"
fn_out_gz="${fn_out}.gz"
rm -f $fn_out $fn_out_zip $fn_out_gz

first="1"
for fn in $(\ls top*); do
    echo "- ${fn}"
    if [ $first == "1" ]; then
        head -n 1 $fn > $fn_out
        first="0"
    fi
    tail -n +2 $fn >> $fn_out
done
zip ${fn_out_zip} $fn_out
echo "Wrote ${fn_out_zip}"
gzip $fn_out
echo "Wrote ${fn_out_gz}"
rm -f $fn_out
