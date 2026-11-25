# STO 🔧

### Auto repair shop management with Go, Next.js, and React Native

STO helps auto repair shops book appointments, manage work orders, and track parts inventory. Customers register their vehicles and track their orders. Technicians manage repairs through vehicle handover, and shop managers oversee orders and maintain catalogs.

The v1 backend supports the full workflow from vehicle registration to handover. The web client and the Android app for technicians are in development. [Current v1 status](stuff/TODO.md)

---

## 🚗 Features

- **Customers** can register vehicles, upload vehicle photos, book service appointments, view order history, pay for repairs, and download a PDF certificate.
- **Technicians** can handle service requests, add parts and services to work orders, update repair stages, and hand vehicles back to customers.
- **Shop managers** can view the order dashboard, search and filter orders, and manage parts and service catalogs.

## 🛠️ Tech Stack

### Backend

![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?style=for-the-badge&logo=postgresql&logoColor=white)
![RabbitMQ](https://img.shields.io/badge/RabbitMQ-FF6600?style=for-the-badge&logo=rabbitmq&logoColor=white)
![OpenAPI](https://img.shields.io/badge/OpenAPI-417E1E?style=for-the-badge&logo=openapiinitiative&logoColor=white)

### Web and Mobile

![Next.js](https://img.shields.io/badge/Next.js-000000?style=for-the-badge&logo=nextdotjs&logoColor=white)
![TypeScript](https://img.shields.io/badge/TypeScript-3178C6?style=for-the-badge&logo=typescript&logoColor=white)
![React Native](https://img.shields.io/badge/React_Native-20232A?style=for-the-badge&logo=react&logoColor=61DAFB)
![Expo](https://img.shields.io/badge/Expo-000020?style=for-the-badge&logo=expo&logoColor=white)

### Infrastructure

![Docker Compose](https://img.shields.io/badge/Docker_Compose-2496ED?style=for-the-badge&logo=docker&logoColor=white)
![nginx](https://img.shields.io/badge/nginx-009639?style=for-the-badge&logo=nginx&logoColor=white)
![Gotenberg](https://img.shields.io/badge/Gotenberg-PDF-505A66?style=for-the-badge)

## 🏗️ Architecture

nginx forwards `/api/*` requests to Gateway, which authenticates requests to protected routes and routes them to the appropriate service.

| Module | Responsibility |
| --- | --- |
| [Gateway](gateway/) | API routing, JWT validation, and data aggregation for the manager dashboard |
| [Users](users/) | Registration, sign-in, and user accounts |
| [Orders](orders/) | Work orders, repair stages, payments, and certificates |
| [Vehicle](vehicle/) | Vehicle records and photos |
| [Inventory](inventory/) | Parts catalog, stock levels, and stock deductions |
| [Services](services/) | Service catalog |
| [chassis](chassis/) | Shared HTTP, RabbitMQ, and outbox libraries |

Each domain service has its own PostgreSQL database. Orders records the payment when the customer marks the order as paid, then asks Inventory to deduct the parts through RabbitMQ and a transactional outbox. Successful deduction enables vehicle handover and queues the PDF certificate. A stock shortage leaves the payment recorded and lets the technician retry after replenishment.

## 🚀 Local Setup

The backend requires Docker with Compose v2 and Make. On first setup, copy the environment template, then set your own passwords and a randomly generated `JWT_SECRET` in `.env`.

```sh
cp .env.example .env
```

```sh
make up
```

Open `http://sto.localhost/` for the web client. nginx serves the application and proxies API requests under `/api/` to Gateway. To run the web client in development mode with hot reload, you need Node.js and npm.

```sh
npm ci --prefix frontend/web
make web
```

The development web client is available at `http://localhost:3001` and sends API requests through the local nginx instance. Use `make web WEB_PORT=3002` to choose another port. See the [technician app README](frontend/mobile/README.md) for Android setup.

## 📚 Documentation

- [OpenAPI specification](api/openapi.yaml)
- [Architecture decision records](stuff/adr/README.md) and [UML diagrams](stuff/uml/)
- [Setup, verification, and development commands](stuff/RUNNING.md)
- [Development plan](stuff/PLAN.md) and [open tasks](stuff/TODO.md)
