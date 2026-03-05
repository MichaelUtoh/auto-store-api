# Connecting to the API from Flutter

When your Flutter app runs in an **iOS Simulator**, **Android Emulator**, or on a **physical device**, `localhost` refers to that device—not the machine where the API is running. So `http://localhost:8089` will fail with "Connection refused".

## Fix: Use the correct base URL

### Option 1: Your computer’s IP (works for simulator, emulator, and physical device)

1. **Find your Mac’s IP** (same Wi‑Fi as the device/simulator):
   ```bash
   ipconfig getifaddr en0
   ```
   (Use `en1` or another interface if `en0` has no address.)

2. **Use that IP in the Flutter app**, e.g.:
   ```text
   http://192.168.1.5:8089
   ```
   Replace with the IP you got from step 1.

3. Ensure the **API is listening on all interfaces** (this repo does: `0.0.0.0:8089`) and, if using Docker, that the port is published (`-p 8089:8089`).

### Option 2: Android Emulator only

From the **Android Emulator**, the host machine’s `localhost` is available at:

```text
http://10.0.2.2:8089
```

Use this as the API base URL when running the Flutter app on the Android emulator.

### Option 3: Flutter on the same Mac (e.g. Chrome / macOS desktop)

If the app runs on the **same machine** as the API (e.g. `flutter run -d chrome` or macOS desktop), then:

```text
http://localhost:8089
```

will work.

## Summary

| Where Flutter runs        | API base URL to use        |
|---------------------------|----------------------------|
| iOS Simulator             | `http://<your-mac-ip>:8089` |
| Android Emulator          | `http://10.0.2.2:8089` or `http://<your-mac-ip>:8089` |
| Physical phone/tablet     | `http://<your-mac-ip>:8089` (same Wi‑Fi) |
| Chrome / macOS (same Mac) | `http://localhost:8089`    |

## Checklist

- [ ] API is running (`go run ./cmd/api` or `docker compose up`).
- [ ] If Docker: port is published (`docker-compose.yml` has `ports: - "8089:8089"`).
- [ ] API listens on `0.0.0.0` (this codebase uses `0.0.0.0:8089`).
- [ ] Flutter app uses the URL that matches where it runs (see table above).
- [ ] Firewall on the Mac allows incoming connections on port 8089 if using IP from another device.
