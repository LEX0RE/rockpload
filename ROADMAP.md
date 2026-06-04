# 🚀 Rockpload Roadmap

## Vision
Rockpload aims to become a powerful multi-platform Rocket League replay manager:
- Upload replays to multiple services simultaneously
- Track multiple accounts and matches in real-time
- Provide live and post-game statistics


## v1.1.0 – Multi Upload System
> Inspired by Ballchasing-like workflow

- [X] Create "Upload Provider" interface
- [X] Implement Ballchasing provider
- [X] Add support for multiple simultaneous uploads
- [X] Queue system for uploads
- [X] Retry mechanism on failure

---

## v1.2.0 – Multi Account Tracking
> Manage multiple Rocket League profiles

- [X] Support multiple account identifiers
- [X] Track replays per account
- [X] Add account config system
- [X] Auto-detect active account

---

## v1.3.0 – Rocket League supervisor
> Handle Rocket League executable for upload

- [X] Auto Upload on Rocket League close
- [X] Prevent Local Account used on Rocket League to be disconnected

---

## v1.4.0 – Live Match Tracking
> Real-time match monitoring

- [ ] Detect active match from StatsAPI
- [ ] Parse match events in real time
- [ ] Send live stats to API
