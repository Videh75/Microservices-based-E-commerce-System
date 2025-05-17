# Go E-commerce Microservices (Ongoing Project)

This is an ongoing project to build a secure, scalable, and modular **E-commerce backend system** using a **microservices-based architecture**. The system is developed in **Go** and leverages modern tools like **gRPC**, **GraphQL**, **Docker**, **PostgreSQL**, and **Elasticsearch**.

## Overview

The application is divided into distinct services:
- `account`: Manages user authentication and profile
- `catalog`: Handles product listings and search
- `order`: Manages orders and transactions

Each service is containerized using Docker and communicates efficiently over **gRPC**. A centralized **GraphQL gateway** provides a unified API interface to clients, enabling flexible querying and streamlined frontend integration.

---

## Tech Stack

- **Go** – Core backend language for all microservices
- **gRPC** – Fast and type-safe service-to-service communication
- **GraphQL** – Centralized API gateway
- **Docker** – Containerization of services
- **Docker Compose** – Local orchestration
- **PostgreSQL** – Relational data storage
- **Elasticsearch** – Full-text search capabilities for catalog

---

## Key Features

- **Microservices Architecture:** Account, Catalog, and Order services, each with clear boundaries.
- **gRPC Communication:** Enables high-performance and efficient communication.
- **GraphQL Gateway:** Serves as the single API entry point for all clients.
- **Containerization with Docker:** Simplifies deployment and testing of individual services.
- **Database Integration:**
  - `PostgreSQL` for structured relational data.
  - `Elasticsearch` for product search in the catalog.
- **Docker Compose:** Streamlines local development by orchestrating all services.
