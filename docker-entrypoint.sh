#!/usr/bin/env bash
set -Eeuo pipefail

mkdir -p /opt/vohive/data /opt/vohive/logs

# The modem's kernel drivers belong to the NAS host. The QDC507 exposes
# serial interfaces 0-3 and its QMI control interface as interface 4. UGREEN
# presents /sys read-only unless it is explicitly bind-mounted by Compose, so
# this setup must be privileged and mount the host's /sys read-write.
driver_log=/opt/vohive/logs/driver-init.log
{
  echo "[$(date -Is)] initializing QDC507 USB drivers"

  # Register QMI before loading option so interface 1.4 can be claimed by
  # qmi_wwan before the serial driver claims every interface.
  if command -v modprobe >/dev/null 2>&1; then
    modprobe cdc_wdm 2>/dev/null || true
    modprobe qmi_wwan 2>/dev/null || true
  fi
  if [[ -w /sys/bus/usb/drivers/qmi_wwan/remove_id ]]; then
    printf '2ca3 4006\n' > /sys/bus/usb/drivers/qmi_wwan/remove_id 2>/dev/null || true
  fi
  if [[ -w /sys/bus/usb/drivers/qmi_wwan/new_id ]]; then
    printf '2ca3 4006 0 2c7c 0125\n' > /sys/bus/usb/drivers/qmi_wwan/new_id 2>/dev/null || true
  fi

  # Load the serial driver after registering the QMI match.
  if command -v modprobe >/dev/null 2>&1; then
    modprobe option 2>/dev/null || true
  fi
  if [[ -w /sys/bus/usb-serial/drivers/option1/new_id ]]; then
    printf '2ca3 4006\n' > /sys/bus/usb-serial/drivers/option1/new_id 2>/dev/null || true
  fi

  # If the host/udev already attached option1 to interface 1.4, release only
  # that interface and bind it to qmi_wwan. USB enumeration can lag container
  # startup, so retry for up to 30 seconds instead of racing the hotplug event.
  bind_qmi_interface() {
    for vendor_file in /sys/bus/usb/devices/*/idVendor; do
      device_dir=${vendor_file%/idVendor}
      [[ -r "$device_dir/idProduct" ]] || continue
      [[ $(<"$vendor_file") == 2ca3 ]] || continue
      [[ $(<"$device_dir/idProduct") == 4006 ]] || continue

      interface="${device_dir}:1.4"
      [[ -d "$interface" ]] || continue
      current_driver=$(basename "$(readlink -f "$interface/driver" 2>/dev/null)" 2>/dev/null || true)
      echo "QDC507 interface=${interface##*/} driver=${current_driver:-none}"
      if [[ "$current_driver" != qmi_wwan ]]; then
        if [[ -n "$current_driver" && -w "$interface/driver/unbind" ]]; then
          printf '%s\n' "${interface##*/}" > "$interface/driver/unbind" 2>/dev/null || true
        fi
        if [[ -w /sys/bus/usb/drivers/qmi_wwan/bind ]]; then
          printf '%s\n' "${interface##*/}" > /sys/bus/usb/drivers/qmi_wwan/bind 2>/dev/null || true
        fi
      fi
    done
  }

  for attempt in $(seq 1 30); do
    bind_qmi_interface
    wdm_found=0
    for wdm_node in /dev/cdc-wdm*; do
      [[ -e "$wdm_node" ]] && wdm_found=1
    done
    if (( wdm_found == 1 )); then
      echo "QDC507 QMI control node is ready"
      break
    fi
    [[ "$attempt" -eq 30 ]] || sleep 1
  done
} >>"$driver_log" 2>&1 || true

# The initial binding above covers container startup.  A USB modem, however,
# may be unplugged and reinserted after VoHive is already running (or may take
# longer than the initial 30 seconds to enumerate after a NAS reboot).  In
# that case udev can let the serial driver claim every interface and no
# /dev/cdc-wdm* control node is created.  Keep a tiny watchdog in this same
# privileged container so the QMI control interface is reclaimed without a
# manual container rebuild/restart.
qmi_hotplug_watchdog() {
  while true; do
    wdm_found=0
    for wdm_node in /dev/cdc-wdm*; do
      [[ -e "$wdm_node" ]] && wdm_found=1
    done

    if (( wdm_found == 0 )); then
      {
        echo "[$(date -Is)] QMI control node missing; recovering QDC507 hotplug"
        if command -v modprobe >/dev/null 2>&1; then
          modprobe cdc_wdm 2>/dev/null || true
          modprobe qmi_wwan 2>/dev/null || true
          modprobe option 2>/dev/null || true
        fi
        if [[ -w /sys/bus/usb/drivers/qmi_wwan/new_id ]]; then
          printf '2ca3 4006 0 2c7c 0125\n' > /sys/bus/usb/drivers/qmi_wwan/new_id 2>/dev/null || true
        fi
        if [[ -w /sys/bus/usb-serial/drivers/option1/new_id ]]; then
          printf '2ca3 4006\n' > /sys/bus/usb-serial/drivers/option1/new_id 2>/dev/null || true
        fi
        bind_qmi_interface
      } >>"$driver_log" 2>&1 || true
    fi

    sleep 5
  done
}

proxy_pid=""
if [[ -x /usr/libexec/qmi-proxy ]]; then
  /usr/libexec/qmi-proxy >/opt/vohive/logs/qmi-proxy.log 2>&1 &
  proxy_pid=$!
fi

qmi_hotplug_watchdog &
watchdog_pid=$!

cleanup() {
  if [[ -n "$proxy_pid" ]]; then
    kill "$proxy_pid" 2>/dev/null || true
  fi
  if [[ -n "${watchdog_pid:-}" ]]; then
    kill "$watchdog_pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT INT TERM

exec /opt/vohive/bin/vohive -c /opt/vohive/config/config.yaml
