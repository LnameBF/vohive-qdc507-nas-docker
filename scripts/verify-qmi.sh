#!/usr/bin/env bash
set -Eeuo pipefail

echo '== USB module =='
lsusb | grep -Ei '2ca3:4006|qdc507|baiwang|dji' || true

echo
echo '== Device nodes =='
ls -l /dev/cdc-wdm* /dev/ttyUSB* 2>&1 || true

echo
echo '== Kernel modules =='
lsmod | grep -E 'qmi_wwan|cdc_wdm|option' || true

echo
echo '== Expected result =='
echo 'At least one /dev/cdc-wdm* node must be present for QMI control.'
