# Live API Verification Scenarios

This document tracks the manual and automated verification of resources against the Baffin Bay live API.

## 🟢 Scenario 1: Traffic Configuration (The Big One)
*Target: `baffinbay_traffic_config`*

- [ ] **TC-01: Basic Creation**
    - Create a config with WAF enabled and a single IPv4 frontend.
    - **Verification:** Confirm the resource appears in the Portal with the correct IP and WAF profile.
- [ ] **TC-02: Update Update Backend**
    - Change the backend IP/Port.
    - **Verification:** Ensure the change propagates without recreating the entire resource (if supported by API).
- [x] **TC-03: Schema Validation (The "IP Bug")**
    - Verify that providing `frontend.ip` no longer returns a 400.
    - **Status:** Code fix implemented. Verified via unit tests. Pending Live API verification once rate-limit expires.
- [ ] **TC-05: Protocol Settings Validation**
    - Ensure `protocolSettings` structure is accepted by API.
    - **Status:** Added to schema/mapping. Pending Live API verification.
- [ ] **TC-04: Deletion**
    - Run `terraform destroy`.
    - **Verification:** Confirm the configuration is gone from the Portal.

## 🔵 Scenario 2: Certificate Management
*Target: `baffinbay_certificate` & `baffinbay_ca_bundle`*

- [ ] **CERT-01: Upload Certificate**
    - Upload a standard PEM certificate.
    - **Verification:** Check "Certificates" section in Portal.
- [ ] **CERT-02: CA Bundle Linkage**
    - Upload a CA Bundle and ensure it can be referenced.

## 🟡 Scenario 3: Custom Pages
*Target: `baffinbay_custom_page`*

- [ ] **PAGE-01: Create Challenge Page**
    - Create a custom HTML challenge page.
    - **Verification:** Preview the page in the Portal.

---
*Note: Use `TF_LOG=DEBUG` during these runs to capture raw JSON payloads for Hermes to analyze.*
