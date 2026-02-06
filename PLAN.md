# Migration Plan: Python to Golang

This document outlines the step-by-step plan to convert the `soundcork` service from Python/FastAPI to Golang. To ensure a smooth transition, we will use a **Proxy-First Approach**, where the new Go service acts as a reverse proxy sitting in front of the existing Python implementation.

## Strategy: Reverse Proxy Delegation

1.  **Go Service (Frontend)**: Listens on the primary port (e.g., `:8000`).
2.  **Python Service (Backend)**: Moved to a secondary port (e.g., `:8001`).
3.  **Delegation Logic**:
    *   If Go has an endpoint implemented, it handles the request.
    *   If Go does not recognize the endpoint, it transparently proxies the request to the Python backend.
4.  **Testing Strategy**:
    *   **Unit Tests**: Each ported Go endpoint/module MUST have accompanying unit tests.
    *   **Integration Tests**: Verify the proxy logic and data persistence consistency.
    *   **End-to-End**: Test with actual SoundTouch hardware where possible.

This allows for incremental porting and testing without breaking existing functionality.

---

## Phase 1: Infrastructure & The Proxy (Week 1)

**Goal**: Establish the Go project and the delegation mechanism.

- [x] Initialize Go module (`go mod init github.com/deborahgu/soundcork`).
- [x] Set up a basic web server (using `net/http` and `Chi` router).
- [x] Implement the **Reverse Proxy**:
    - Use `httputil.ReverseProxy` to forward unrecognized requests to `http://localhost:8001`.
- [x] Update `Dockerfile` to a multi-stage build:
    - [x] Stage 1: Build Go binary.
    - [x] Stage 2: Final image containing both Go binary and Python environment.
    - [x] Stage 3: Process manager (like `supervisord` or a simple shell script) to run both.
- [x] Port basic configuration management (Environment variables).
- [x] Establish testing baseline (Add `main_test.go`).

## Phase 2: Core Models & Static Content (Week 1)

**Goal**: Port data structures and simple endpoints.

- [x] Translate `soundcork/model.py` (Pydantic) to Go `structs` with JSON/XML tags.
- [x] Implement the root endpoint (`GET /`) in Go.
- [x] Implement static file serving for `/media`.
- [x] Port `soundcork/constants.py` to Go constants or configuration files.
    - [x] Mirror speaker-related constants (`SpeakerHTTPPort`, etc.) for future automation.

## Phase 3: BMX (Streaming & Service Registry) (Week 2)

**Goal**: Port the service registry and external API integrations.

- [x] Port `soundcork/bmx.py`:
    - [x] Implement TuneIn XML/OPML parsing.
    - [x] Implement `tunein_playback` and `tunein_podcast_info` logic.
- [x] Port `bmx_services.json` logic and `swupdate.xml` serving.
- [x] Implement the `/v1/services` and `/v1/playback` endpoints in Go.
- [x] Implement comprehensive unit tests for all internal packages (`datastore`, `marge`, `bmx`, `constants`).
- [x] Fix potential runtime panics identified during testing.

## Phase 4: Datastore & Marge (Speaker Communication) (Week 3)

**Goal**: Port the persistence layer and complex XML generation.

- [x] Implement the `DataStore` in Go:
    - [x] Filesystem operations (Read/Write XML).
    - [x] Directory structure management (`/data/{account}/devices/...`).
- [x] Port `soundcork/marge.py` logic:
    - [x] XML generation for Presets, Recents, and Account Info.
    - [x] Logic for adding/removing devices.
- [x] Implement Marge endpoints:
    - [x] `/marge/streaming/sourceproviders`
    - [x] `/marge/accounts/{account}/full`
    - [x] `/marge/accounts/{account}/devices/{device}/presets`
- [x] **Checkpoint**: At this stage, almost all functional traffic should be handled by Go.

## Phase 5: Discovery & UPNP (Week 4)

**Goal**: Remove the last dependency on Python.

- [x] Implement UPNP/SSDP discovery in Go (e.g., using `github.com/gesellix/Bose-SoundTouch/pkg/discovery`).
- [x] Port the background task for device discovery (`lifespan` equivalent).
- [x] Refactor endpoint handlers into dedicated Go files (e.g., `handlers_bmx.go`, `handlers_marge.go`).
- [x] Implement comprehensive HTTP handler tests for all Go endpoints and split them into dedicated test files.
- [x] Add GitHub Actions workflow for the Go implementation.
- [ ] Final end-to-end verification with physical speakers.

## Phase 6: Decommissioning (Week 4)

**Goal**: Clean up and optimize.

- [ ] Remove the Python reverse proxy delegation.
- [ ] Delete all `.py` files and `requirements.txt`.
- [ ] Update `Dockerfile` to a single-stage Go build (reducing image size from ~200MB to ~20MB).
- [ ] Final documentation updates.

## Phase 7: Automated Setup & UI (Future)

**Goal**: Simplify the initial configuration of SoundTouch devices.

- [x] Implement an automated "Setup" feature in `soundcork-go`:
    - [x] Create a standalone `setup-speaker.sh` script for easy migration.
    - [x] Integrate SSH/SCP (via Go `ssh` library) into the Go service for a programmatic approach.
    - [x] Auto-discover devices and offer a "Migrate to Soundcork" button.
- [x] Build a basic Web UI for monitoring and managing discovered devices.
- [x] Implement migration summary view with SSH check and config comparison.
    - [x] Improve SSH error reporting and UI feedback for unreachable devices.
- [x] Refactor Web UI to use external HTML file with Go `embed`.
- [x] Implement Remote Services Persistence (stable `remote_services` via trigger file).
- [x] Implement Granular Proxying (per-service choice between Soundcork and Original Upstream).
- [x] Implement Live Device Info retrieval from `:8090/info`.
- [x] Implement Manual Configuration Backup and Viewing UI.
- [x] Implement Custom Target and Proxy Domain settings in UI.
- [x] Improve SSH robustness (legacy ciphers, `ssh-rsa` support, and `rw` filesystem remounting).

---

## Technical Mapping Reference

| Component     | Python Tool             | Go Tool                                             |
|:--------------|:------------------------|:----------------------------------------------------|
| Routing       | FastAPI                 | `net/http` + `Chi`                                  |
| Serialization | Pydantic / ElementTree  | `encoding/json` / `encoding/xml`                    |
| Proxying      | N/A                     | `net/http/httputil`                                 |
| UPNP          | `upnpclient`            | `github.com/gesellix/Bose-SoundTouch/pkg/discovery` |
| XML Parsing   | `xml.etree.ElementTree` | `encoding/xml`                                      |
| Config        | `pydantic-settings`     | `github.com/caarlos0/env`                           |

## Phase 8: Upstream Parity (Feb 2026)

**Goal**: Align Go implementation with recent upstream changes (ETags, Base64, timestamps).

- [x] Align Base64 robustness for BMX (try URL-safe first).
- [x] Align Recents timestamp behavior (preserve `createdOn`).
- [x] Implement ETag handling for Marge endpoints:
    - [x] `GET /marge/streaming/sourceproviders`
    - [x] `GET /marge/accounts/{account}/full`
    - [x] `GET /marge/accounts/{account}/devices/{device}/presets`
    - [x] `POST /accounts/{account}/devices/{device}/recents` (Response header)
    - [x] `GET /marge/updates/soundtouch`
- [x] Respect `If-None-Match` and return `304 Not Modified` where applicable.
- [x] Add unit tests for ETag and `304` behavior.
- [x] Implement `DataStore.Initialize()` for directory bootstrapping.
- [x] Improve error handling for malformed XML inputs in DataStore.
- [ ] Evaluate "Create Account from Device" parity.

## Phase 9: Proxy Instrumentation & Monitoring (Feb 2026)

**Goal**: Complete view on device-upstream interaction for debugging and analysis.

- [x] Implement comprehensive logging for the proxy:
    - [x] Log request/response URLs, methods, status codes.
    - [x] Log headers with default redaction for sensitive fields (e.g., `X-Bose-Token`, `Authorization`).
    - [x] Log request/response bodies.
    - [x] Control logging settings (redact, log body) via Web UI.
- [x] Implement smart body logging:
    - [x] Skip/truncate logging for streaming data or excessively large bodies.
    - [x] Detect `Content-Type` to decide on logging format (text/xml vs binary).
- [x] Add control for redaction and logging verbosity (e.g., via environment variables).
- [x] Ensure all proxy interactions are traceable for troubleshooting (e.g. issues like #129).

---
