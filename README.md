SaaS Social Media Management Platform
This is a comprehensive, multi-tenant social media management platform built with a microservices architecture. It allows users to securely connect their Meta (Facebook/Instagram), TikTok, and Snapchat accounts to a single interface, where they can manage posts, schedule content, and view analytics.

The platform is designed with a SaaS (Software-as-a-Service) model, ensuring strict data isolation between different customer accounts (tenants).

Architecture
The project is divided into three independent Go microservices and a React frontend.

Auth Service (Port 8081):

Handles all user authentication and OAuth flows for social media platforms.

Manages user registration and assigns a unique tenant_id to each new customer.

Generates a JSON Web Token (JWT) containing both the user_id and tenant_id for secure, multi-tenant-aware communication.

Account Service (Port 8082):

Manages all connected social media accounts.

Stores access tokens, refresh tokens, and account details in a PostgreSQL database.

All database operations are scoped by tenant_id, ensuring a customer can only access their own connected accounts.

Post Service (Port 8083):

Handles the creation, scheduling, and management of social media posts.

Stores post content, scheduled times, and status in a PostgreSQL database.

Provides API endpoints for retrieving posts and a mock endpoint for analytics. All data access is filtered by tenant_id.

React Frontend (Port 3000):

A single-page application built with React and Tailwind CSS.

Provides the user-friendly interface for the dashboard, post scheduler, and analytics.

Communicates with the backend microservices using the JWT for authentication.

Getting Started
Follow these steps to set up and run the entire application.

Prerequisites
You'll need the following installed on your machine:

Go (1.18 or newer): For the backend services.

Node.js & npm: For the React frontend.

PostgreSQL: A running database server.

Step 1: Database Setup
Create a new PostgreSQL database (e.g., smm_platform).

Set your DATABASE_URL as an environment variable in your terminal. This is crucial for all backend services to connect to the database.

Example (Linux/macOS):

export DATABASE_URL="host=localhost port=5432 user=your_user password=your_password dbname=smm_platform sslmode=disable"

Example (Windows PowerShell):

$env:DATABASE_URL="host=localhost port=5432 user=your_user password=your_password dbname=smm_platform sslmode=disable"

The database tables will be automatically created by each service when it starts.

Step 2: Social Media API Credentials
Register your application with the developer portals for each platform to get your credentials. You must replace the placeholder values in the main.go file of the auth-service with your actual keys.

Meta for Developers: https://developers.facebook.com/

TikTok for Developers: https://developers.tiktok.com/

Snapchat Marketing API: https://marketingapi.snapchat.com/

Step 3: Run the Backend Services
Open three separate terminal windows and run each service in its dedicated directory.

Auth Service (auth-service/)

cd auth-service
go mod tidy
go run main.go

This service will run on http://localhost:8081.

Account Service (account-service/)

cd account-service
go mod tidy
go run main.go

This service will run on http://localhost:8082.

Post Service (post-service/)

cd post-service
go mod tidy
go run main.go

This service will run on http://localhost:8083.

Step 4: Run the Frontend
Open a fourth terminal window for the React application.

cd frontend
npm install
npm start

The frontend will open on http://localhost:3000. You can now use the login buttons to begin.

Future Improvements
Implement the content library and engagement management features.

Integrate real-time data fetching and API calls for posts and analytics in the Post Service.

Add a dedicated Billing Service to manage subscriptions and payment gateways.

Implement a dashboard for administrators to manage tenants and monitor system health.
