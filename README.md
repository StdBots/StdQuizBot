# 🎲 StdQuizBot — Enterprise Quiz & Multiplayer Trivia Tournament Bot

<p align="center">
  <img src="https://graph.org/file/00ea4effe5d2dfbb8d5be.jpg" alt="StdQuizBot Banner" width="450"/>
</p>

<p align="center">
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Language-Go%201.22+-00ADD8?style=for-the-badge&logo=go" alt="Go"/></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-AGPL%20v3-blue?style=for-the-badge" alt="License"/></a>
  <a href="https://t.me/STDBOTS"><img src="https://img.shields.io/badge/Channel-%40STDBOTS-2CA5E0?style=for-the-badge&logo=telegram" alt="Telegram Channel"/></a>
  <a href="https://deepanshu.in"><img src="https://img.shields.io/badge/Author-STD%20DEEPANSHU-FF4500?style=for-the-badge" alt="Author"/></a>
</p>

<p align="center">
  <a href="https://dashboard.heroku.com/new?template=https://github.com/StdBots/StdQuizBot">
    <img src="https://img.shields.io/badge/Deploy%20To%20Heroku-7056bf?style=for-the-badge&logo=heroku" alt="Deploy to Heroku"/>
  </a>
  <a href="https://railway.app/template/new?template=https://github.com/StdBots/StdQuizBot">
    <img src="https://img.shields.io/badge/Deploy%20On%20Railway-0B0D0E?style=for-the-badge&logo=railway" alt="Deploy on Railway"/>
  </a>
</p>

---

## ⚡ Overview

**StdQuizBot** is an enterprise-grade Telegram Quiz, Trivia, and Multiplayer Tournament engine engineered in **Go (Golang)** with **MongoDB** persistence.

Inspired by Telegram's official `@QuizBot`, it elevates trivia gameplay by solving previous Python bot desync issues with **flawless dual-mode support** (both solo private DM and live group tournament sessions), native Telegram Quiz Poll animations (green checkmark, red cross, confetti), millisecond speed scoring, and **pure Go HD Victory Podium Banners (<15ms per card without external graphics dependencies)**.

Developed by **[STD DEEPANSHU](https://deepanshu.in)** as part of the **[STD BOTS Ecosystem](https://t.me/STDBOTS)**.

---

## ✨ Features

- 🎯 **Native Telegram Quiz Polls:**
  - Dispatches official Telegram `type="quiz"` polls with correct answer highlights and explanatory text.
  - Native animated feedback: green checks, red crosses, and celebratory confetti.
- 🎮 **Dual-Mode Gameplay:**
  - **Solo Private Play (DM):** Play quizzes individually with instant next-question progression as soon as an option is selected.
  - **Multiplayer Group Tournaments:** Add bot to any group to launch a live lobby. Group members join with real-time `[ N ✅ I am ready! ]` button counters. Tournaments auto-start when 2+ players are ready.
- ⚡ **Speed-Based Scoring System:**
  - Base points: `100 points` for each correct answer.
  - Speed bonus: up to `+50 extra points` for fast responses (calculated dynamically in tenths of a second).
- 🏆 **Pure Go HD Victory Podium Banners:**
  - 1000x640 HD PNG podium victory banner generated in memory (<15ms).
  - Features Olympic podium cards for **🥇 Champion (Gold)**, **🥈 Runner-up (Silver)**, and **🥉 3rd Place (Bronze)** with scores, accuracy, and average response times.
- ✏️ **Interactive Quiz Creation & Management:**
  - Direct question poll intake using Telegram's native poll creator.
  - Support for intro text/media sent prior to a question.
  - Full editor: Add questions, delete questions, rename, change timer (10s to 5m), and set shuffle modes (`Shuffle All`, `Only Questions`, `Only Answers`, `No Shuffle`).
- 📤 **Inline Quiz Sharing:**
  - Share quizzes in any chat or channel using inline mode (`@StdQuizBot <quiz_id>`).
- 📢 **Admin Broadcast & Analytics Suite:**
  - High-speed concurrent message broadcaster with worker pools and automatic 429 FloodWait handling.
  - Real-time performance analytics (`/stats`) monitoring memory alloc, goroutines, and active quiz velocity.
- 🔒 **7-Layer Forced Credit Protection:**
  - AGPL-3.0 attribution, zero-width steganographic watermarks, tamper-detection, and telemetry monitoring for STD DEEPANSHU & @STDBOTS.

---

## 🚀 One-Click Deployments

### 🟣 Deploy to Heroku
Click the button below to deploy your instance to Heroku in under 60 seconds:

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://dashboard.heroku.com/new?template=https://github.com/StdBots/StdQuizBot)

### 🚂 Deploy on Railway
Click the button below to deploy on Railway with container support:

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/template/new?template=https://github.com/StdBots/StdQuizBot)

### 🖥️ 1-Command VPS Deployment (Linux / Ubuntu / Debian)
Run this single command on your VPS as root:
```bash
curl -fsSL https://raw.githubusercontent.com/StdBots/StdQuizBot/main/scripts/install_vps.sh | bash
```

---

## 🐳 Docker & Manual VPS Setup

```bash
# 1. Clone repository
git clone https://github.com/StdBots/StdQuizBot.git
cd StdQuizBot

# 2. Configure environment
cp .env.example .env
nano .env

# 3. Start with Docker Compose
docker compose up -d --build
```

---

## ⚙️ Environment Variables

| Variable | Description | Required | Default |
| :--- | :--- | :--- | :--- |
| `BOT_TOKEN` | Telegram Bot Token from [@BotFather](https://t.me/BotFather) | **Yes** | — |
| `OWNER_ID` | Telegram Numeric User ID of the Bot Owner | **Yes** | `7394590844` |
| `MONGO_URI` | MongoDB Connection String (Atlas or Local) | **Yes** | `mongodb://localhost:27017/stdquizbot` |
| `DB_NAME` | Database Name | No | `stdquizbot` |
| `FORCE_SUB_CHANNEL` | Channel username for force subscribe (without `@`) | No | `StdBots` |
| `LOG_CHANNEL_ID` | Channel ID for audit & alert forwarding | No | `0` |
| `ENV` | Environment (`production` / `development`) | No | `production` |

---

## ⌨️ Bot Commands

| Command | Scope | Description |
| :--- | :--- | :--- |
| `/start` | Everywhere | Welcome menu, open shared quiz cards, or start group lobby |
| `/newquiz` | DM | Start interactive quiz creation wizard |
| `/quizzes` or `/myquizzes` | DM | View, edit, and manage all your created quizzes |
| `/quiz <id>` | Everywhere | Launch quiz card in DM or open tournament lobby in group |
| `/stop` | Everywhere | Cancel an ongoing active quiz session |
| `/undo` | DM | Revert the last question while creating a quiz |
| `/skip` | DM | Skip the optional quiz description step |
| `/done` | DM | Finish adding questions and proceed to timer configuration |
| `/help` | Everywhere | Complete guide on playing, hosting, and creating quizzes |
| `/stats` | Owner | Real-time performance analytics, goroutines, and memory alloc |
| `/broadcast` | Owner | High-speed concurrent announcement engine across all quiz players |

---

## 🔒 7-Layer Forced Credit Protection

This repository is distributed under the **GNU Affero General Public License v3 (AGPL-3.0)** with strict attribution requirements:
1. **Source Code Header:** Preserved in every source and header file.
2. **Obfuscated Strings:** Dev identity encoded via multi-layer string obfuscation.
3. **Invisible Steganographic Watermarks:** Injected into Telegram messages using zero-width Unicode characters (`\u200B`, `\u200C`, `\u200D`).
4. **Runtime Integrity Guard:** Verifies developer signatures on startup; halts execution if tampered.
5. **UI & Banner Footers:** Active hyperlinks on all interactive keyboards and generated HD images.
6. **Telemetry Heartbeat:** Non-blocking telemetry reporting fork statuses.
7. **Strict AGPL-3.0 Additional Terms:** Commercial or modified instances MUST display "Powered by STD BOTS (@STDBOTS)".

---

## 👨‍💻 Developer & Support

- **Lead Architect:** [STD DEEPANSHU](https://deepanshu.in)
- **Telegram Channel:** [@STDBOTS](https://t.me/STDBOTS)
- **Personal Handle:** [@STD_DEEPANSHU](https://t.me/STD_DEEPANSHU)
- **Website:** [https://deepanshu.in](https://deepanshu.in)

Copyright (c) 2024-2026 STD DEEPANSHU & STD BOTS. All Rights Reserved.
