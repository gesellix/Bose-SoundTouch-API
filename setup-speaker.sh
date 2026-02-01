#!/bin/bash

# Simple script to update SoundTouch speaker configuration
# Based on Soundcork issue #59

SERVER_URL=${1:-"http://$(hostname):8000"}
DEVICE_IP=$2

if [ -z "$DEVICE_IP" ]; then
    echo "Usage: $0 <SERVER_BASE_URL> <DEVICE_IP>"
    echo "Example: $0 http://192.168.1.100:8000 192.168.1.150"
    exit 1
fi

cat << XML > SoundTouchSdkPrivateCfg.xml
<?xml version="1.0" encoding="utf-8"?>
<SoundTouchSdkPrivateCfg>
  <margeServerUrl>${SERVER_URL}/marge</margeServerUrl>
  <statsServerUrl>${SERVER_URL}</statsServerUrl>
  <swUpdateUrl>${SERVER_URL}/updates/soundtouch</swUpdateUrl>
  <usePandoraProductionServer>true</usePandoraProductionServer>
  <isZeroconfEnabled>true</isZeroconfEnabled>
  <saveMargeCustomerReport>false</saveMargeCustomerReport>
  <bmxRegistryUrl>${SERVER_URL}/bmx/registry/v1/services</bmxRegistryUrl>
</SoundTouchSdkPrivateCfg>
XML

echo "Generated SoundTouchSdkPrivateCfg.xml with server $SERVER_URL"
echo "Attempting to upload to $DEVICE_IP..."

# Using flags recommended in issue #59 and previous README updates
# Note: -O is for legacy SCP protocol, -oHostKeyAlgorithms=+ssh-dss or ssh-rsa for old servers
scp -O \
    -o PreferredAuthentications=password \
    -o PubkeyAuthentication=no \
    -o HostKeyAlgorithms=+ssh-dss,ssh-rsa \
    -o StrictHostKeyChecking=no \
    SoundTouchSdkPrivateCfg.xml root@${DEVICE_IP}:/opt/Bose/etc/SoundTouchSdkPrivateCfg.xml

if [ $? -eq 0 ]; then
    echo "Upload successful. Rebooting speaker..."
    ssh -o HostKeyAlgorithms=+ssh-dss,ssh-rsa \
        -o StrictHostKeyChecking=no \
        root@${DEVICE_IP} "rw && reboot"
    echo "Speaker at $DEVICE_IP is rebooting."
else
    echo "Failed to upload configuration to $DEVICE_IP."
    exit 1
fi
