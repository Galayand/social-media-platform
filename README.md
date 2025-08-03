# 📱 SaaS Social Media Management Platform

A comprehensive, **multi-tenant** social media management platform built with a **microservices architecture**. Users can securely connect their **Meta (Facebook/Instagram)**, **TikTok**, and **Snapchat** accounts to a single interface to manage posts, schedule content, and view analytics.

Designed as a **SaaS (Software-as-a-Service)** platform, it ensures **strict data isolation** between customer accounts (tenants).

---

## 🧱 Architecture

The platform is divided into **three independent Go microservices** and a **React frontend**:

### 🔐 Auth Service (Port `8081`)
- Handles user authentication and OAuth flows.
- Manages user registration and assigns a unique `tenant_id`.
- Issues JWT tokens containing `user_id` and `tenant_id`.

### 🧾 Account Service (Port `8082`)
- Manages connected social media accounts.
- Stores tokens and account details in a PostgreSQL database.
- Scopes all operations by `tenant_id`.

### 📝 Post Service (Port `8083`)
- Manages the creation, scheduling, and status of social media posts.
- Includes endpoints for retrieving posts and mock analytics.
- Filters all data access by `tenant_id`.

### 🎨 React Frontend (Port `3000`)
- Single-page application built with **React** and **Tailwind CSS**.
- Dashboard for post management and analytics.
- Uses JWT tokens to communicate with backend services.

---

## 🚀 Getting Started

Follow these steps to set up and run the application locally.

### ✅ Prerequisites

Make sure the following are installed:

- **Go (1.18+)** – For backend services  
- **Node.js & npm** – For the frontend  
- **PostgreSQL** – Running database server  

---

## 🗂 Step 1: Database Setup

1. **Create a PostgreSQL database** (e.g., `smm_platform`).
2. **Set the environment variable `DATABASE_URL`**:

**Linux/macOS:**
```bash
export DATABASE_URL="host=localhost port=5432 user=your_user password=your_password dbname=smm_platform sslmode=disable"
```

**Windows PowerShell:**
```powershell
$env:DATABASE_URL="host=localhost port=5432 user=your_user password=your_password dbname=smm_platform sslmode=disable"
```

Each service will auto-create its required tables on startup.

---

## 🔑 Step 2: Social Media API Credentials

Register your app with each platform to get API credentials, then replace the placeholder values in `main.go` in the **Auth Service**:

- [Meta for Developers](https://developers.facebook.com/)
- [TikTok for Developers](https://developers.tiktok.com/)
- [Snapchat Marketing API](https://marketingapi.snapchat.com/)

---

## 🧩 Step 3: Run Backend Services

Open **three terminal windows**, one for each service:

### Auth Service
```bash
cd auth-service
go mod tidy
go run main.go
```
➡️ Runs on: `http://localhost:8081`

---

### Account Service
```bash
cd account-service
go mod tidy
go run main.go
```
➡️ Runs on: `http://localhost:8082`

---

### Post Service
```bash
cd post-service
go mod tidy
go run main.go
```
➡️ Runs on: `http://localhost:8083`

---

## 💻 Step 4: Run the Frontend

Open a **fourth terminal window**:

```bash
cd frontend
npm install
npm start
```

➡️ Opens on: `http://localhost:3000`  
Use the login buttons to begin authentication.

---

## 🔮 Future Improvements

- 📂 Add a **Content Library** and **Engagement Manager**
- 📊 Real-time API integration for posts & analytics
- 💳 Dedicated **Billing Service** (subscriptions & payments)
- 🛠 Admin dashboard for tenant management and monitoring

