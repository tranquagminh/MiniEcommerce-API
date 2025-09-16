# MiniEcommerce-API Guide

Welcome to the MiniEcommerce-API project! Follow this guide to set up and run the project on your local machine.

## Prerequisites

- **Docker**: Make sure Docker is installed and running on your desktop. [Download Docker](https://www.docker.com/products/docker-desktop/)
- **Go**: Install Go from the official website. [Download Go](https://go.dev/dl/)

## Getting Started

1. **Clone the repository** (if you haven't already):

    ```
    git clone <your-repo-url>
    cd MiniEcommerce-API
    ```

2. **Start the services using Docker Compose**:

    ```
    docker-compose up -d
    ```

    This command will start all necessary services in the background.

3. **Rebuild the project after code changes**:

    If you make changes to the code and want to rebuild the containers, run:

    ```
    docker-compose build --no-cache
    ```

    Then restart the services:

    ```
    docker-compose up -d
    ```

## Additional Notes

- Make sure your Docker daemon is running before executing the above commands.
- For any issues, check the logs using:

  ```
  docker-compose logs
  ```

Happy coding!