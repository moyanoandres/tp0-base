"""
Docker Compose Generator

This script generates a Docker Compose file with a specified number of client services \
based on a template YAML file.

Usage:
    python3 compose_maker.py [number_of_clients]

Arguments:
    number_of_clients (int, optional): The number of client services to add to the Docker Compose file.
                                       Defaults to 1 if not provided.

Example:
    python script.py 3

"""

import yaml
import argparse

filename = 'docker-compose-dev.yaml'

CLIENTLESS_YAML_DATA = """
version: '3.9'
name: tp0
services:
  server:
    container_name: server
    image: server:latest
    entrypoint: python3 /main.py
    environment:
      - PYTHONUNBUFFERED=1
      - LOGGING_LEVEL=DEBUG
    networks:
      - testing_net

networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24
"""

def create_compose_file(data, filename, number_of_clients):
    services = data.get('services', {})
    for i in range(1, number_of_clients + 1):
        client_name = f"client{i}"
        services[client_name] = {
            "container_name": client_name,
            "image": "client:latest",
            "entrypoint": "/client",
            "environment": [
                f"CLI_ID={i}",
                "CLI_LOG_LEVEL=DEBUG"
            ],
            "networks": ["testing_net"],
            "depends_on": ["server"]
        }
    
    data['services'] = services

    with open(filename, "w") as file:
        yaml.dump(data, file)


if __name__ == "__main__":
  parser = argparse.ArgumentParser(description="Create Docker Compose file with specified number of clients.")
  parser.add_argument("number_of_clients", type=int, nargs='?', default=1, help="Number of client services to add (default: 1)")
  args = parser.parse_args()

  clientless_yaml_data = yaml.safe_load(CLIENTLESS_YAML_DATA)
  
  create_compose_file(clientless_yaml_data, filename, args.number_of_clients)

  print(f"YAML file '{filename}' supporting {args.number_of_clients} client(s) has been created successfully.")