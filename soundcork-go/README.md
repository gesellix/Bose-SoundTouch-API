# soundcork-go
Intercept API for Bose SoundTouch after they turn off the servers

## Status

This project is the Golang implementation of soundcork. It is currently in the process of replacing the original Python implementation.

## Background
[Bose has announced that they are shutting down the servers for the SoundTouch system in February, 2026. ](https://www.bose.com/soundtouch-end-of-life) When those servers go away, certain network-based functionality currently available to SoundTouch devices will stop working.

This is an attempt to reverse-engineer those servers so that users can continue to use the full set of SoundTouch functionality after Bose shuts the official servers down.

### Context

[As described here](https://flarn2006.blogspot.com/2014/09/hacking-bose-soundtouch-and-its-linux.html), it is possible to access the underlying speaker by creating a USB stick with an empty file called `remote_services` and then booting the SoundTouch with the USB stick plugged in to the USB port in the back. From there we can then telnet (port 23, some older blog articles mention 17000) or ssh (but the ssh server running is fairly old) over and log in as root (no password).

Once logged into the speaker, you can go to `/opt/Bose/etc` and look at the file `SoundTouchSdkPrivateCfg.xml`:

```xml
<?xml version="1.0" encoding="utf-8"?>
<SoundTouchSdkPrivateCfg>
  <margeServerUrl>https://streaming.bose.com</margeServerUrl>
  <statsServerUrl>https://events.api.bosecm.com</statsServerUrl>
  <swUpdateUrl>https://worldwide.bose.com/updates/soundtouch</swUpdateUrl>
  <usePandoraProductionServer>true</usePandoraProductionServer>
  <isZeroconfEnabled>true</isZeroconfEnabled>
  <saveMargeCustomerReport>false</saveMargeCustomerReport>
  <bmxRegistryUrl>https://content.api.bose.io/bmx/registry/v1/services</bmxRegistryUrl>
</SoundTouchSdkPrivateCfg>
```

Assumingly all four servers listed there will be shut down. From testing, the `marge` server is necessary for basic network functionality, and the `bmx` server seems to be required for TuneIn radio at least. The stats and swUpdate addresses don't seem to be necessary for the speaker to function.

## Running, testing, and installing soundcork (Go version)

### Installing

This has been written and tested with Go 1.25.

1. Ensure you have Go installed on your system.
2. Clone the repository and navigate to the `soundcork-go` directory.
3. Build the binary:
   ```sh
   go build -o soundcork ./soundcork-go
   ```

### Running

#### Using Docker (Recommended)

The root directory contains a `Dockerfile` that builds and runs the service.

1. Build the Docker image:
   ```sh
   docker build -t soundcork .
   ```
2. Run the container:
   ```sh
   docker run -d \
     -p 8000:8000 \
     -e PORT=8000 \
     -e BASE_URL="http://your-host-ip:8000" \
     -v /path/to/your/data:/data \
     --name soundcork \
     soundcork
   ```

#### Running the Binary Directly

```sh
export PORT=8000
export BASE_URL="http://localhost:8000"
export DATA_DIR="./data"
./soundcork
```

### Configuration

The following environment variables are supported:

| Variable | Description | Default |
| :--- | :--- | :--- |
| `PORT` | The port the server listens on | `8000` |
| `BIND_ADDR` | The address to bind to | (all interfaces) |
| `BASE_URL` | The public URL of the server | `http://localhost:8000` |
| `DATA_DIR` | Directory for storing device data | `data` |
| `MEDIA_DIR` | Directory for static media files | `soundcork/media` |
| `PYTHON_BACKEND_URL` | URL for the legacy Python backend (if used as proxy) | `http://localhost:8001` |

### Setting your SoundTouch device to use the soundcork server

For purposes of this example, let's say that you've set up a soundcork server on your local server available via hostname `soundcork.local.example.com` and running on port 8000. Let's also say that you want a data dir at `/home/soundcork/db`.

Start the server with the appropriate environment variables:

```sh
PORT=8000 BASE_URL="http://soundcork.local.example.com:8000" DATA_DIR="/home/soundcork/db" ./soundcork
```

### Configuring the speaker

Once a soundcork server is running, the next step is to configure your SoundTouch device to run using soundcork instead of the Bose servers. Follow the steps below to gain access and configure the speaker.

#### Step 1: Prepare USB Drive
- Insert USB stick into computer.
- Create remote services file: `touch /path/to/mounted/usb/root-directory/remote_services`

#### Step 2: Connect to Device
- Insert USB stick into SoundTouch device.
- Restart device (unplug power, plug it back in).

#### Step 3: Access Device via SSH
After restart, SSH access is enabled. You can connect via telnet or SSH.

**Via SSH (Recommended):**
```sh
ssh -oHostKeyAlgorithms=ssh-rsa root@<device-ip>
```
*Note: No password required for root access. The `-oHostKeyAlgorithms=ssh-rsa` option is often necessary because the speaker's SSH server is old.*

**Via Telnet:**
```sh
telnet <device-ip> 23
```
Log in as user `root` (no password).

#### Step 4: Check Current Configuration
Once logged in, you can view the current configuration:
```sh
cat /opt/Bose/etc/SoundTouchSdkPrivateCfg.xml
```
Note the URLs for streaming, stats, software updates, and BMX registry.

#### Step 5: Setting up the soundcork server data
The general layout of the soundcork data directory (e.g., `/home/soundcork/db`) is:

```
/home/soundcork/db/{account}/Presets.xml
/home/soundcork/db/{account}/Recents.xml
/home/soundcork/db/{account}/Sources.xml
/home/soundcork/db/{account}/devices/{deviceid}/DeviceInfo.xml
```

1. **Get IDs**: Access `http://192.168.1.158:8090/info` to find your `deviceID` and `margeAccountUUID`.
2. **Create Directories**:
   ```sh
   mkdir -p /home/soundcork/db/{account}/devices/{deviceid}
   ```
3. **Fetch Data**: Fetch `Presets.xml`, `Recents.xml`, and `DeviceInfo.xml` from the speaker's web API (port 8090) and save them to the corresponding locations in your data directory.
4. **Fetch Sources**: `Sources.xml` must be fetched from the device via `scp` or USB:
   ```sh
   # On the speaker
   cd /mnt/nv/BoseApp-Persistence/1/
   scp Sources.xml user@host:/home/soundcork/db/{account}/
   ```

*Note on `Sources.xml`*: If sources don't have an `id` attribute, soundcork will assign them automatically, but you can manually add one for stability: `<source displayName="AUX IN" id="123456" ...>`.

#### Step 6: Configure the Bose speaker to use the soundcork server

You can configure the speaker either by editing the file directly on the device using `vi`, or by downloading it, modifying it locally, and uploading it back.

**Option A: Edit directly on the device**

1. Telnet or SSH into the speaker.
2. Set the filesystem to read-write: `rw`.
3. Edit `/opt/Bose/etc/SoundTouchSdkPrivateCfg.xml` using `vi`.
4. Update the URLs to point to your soundcork server (see below).
5. Reboot the speaker: `reboot`.

**Option B: Download, modify, and upload (using scp)**

1. Download the file from the speaker:
   ```sh
   scp -oHostKeyAlgorithms=ssh-rsa root@<device-ip>:/opt/Bose/etc/SoundTouchSdkPrivateCfg.xml ./SoundTouchSdkPrivateCfg.xml
   ```
2. Modify the file locally using your favorite text editor.
3. SSH into the speaker and set the filesystem to read-write:
   ```sh
   ssh -oHostKeyAlgorithms=ssh-rsa root@<device-ip> "rw"
   ```
4. Upload the modified file back to the speaker:
   ```sh
   scp -oHostKeyAlgorithms=ssh-rsa ./SoundTouchSdkPrivateCfg.xml root@<device-ip>:/opt/Bose/etc/SoundTouchSdkPrivateCfg.xml
   ```
5. Reboot the speaker:
   ```sh
   ssh -oHostKeyAlgorithms=ssh-rsa root@<device-ip> "reboot"
   ```

**Option C: Use the automated setup script**

A helper script `setup-speaker.sh` is provided in the root directory to automate the generation and upload of the configuration file.

1. Ensure your soundcork server is running.
2. Run the script with your server's base URL and the speaker's IP:
   ```sh
   ../setup-speaker.sh http://your-host-ip:8000 <device-ip>
   ```
   The script will generate the XML, upload it using `scp`, set the speaker to read-write mode, and trigger a reboot.

**Target Configuration:**

Update the URLs in `SoundTouchSdkPrivateCfg.xml` as follows:

```xml
<?xml version="1.0" encoding="utf-8"?>
<SoundTouchSdkPrivateCfg>
  <margeServerUrl>http://soundcork.local.example.com:8000/marge</margeServerUrl>
  <statsServerUrl>http://soundcork.local.example.com:8000</statsServerUrl>
  <swUpdateUrl>http://soundcork.local.example.com:8000/updates/soundtouch</swUpdateUrl>
  <usePandoraProductionServer>true</usePandoraProductionServer>
  <isZeroconfEnabled>true</isZeroconfEnabled>
  <saveMargeCustomerReport>false</saveMargeCustomerReport>
  <bmxRegistryUrl>http://soundcork.local.example.com:8000/bmx/registry/v1/services</bmxRegistryUrl>
</SoundTouchSdkPrivateCfg>
```
