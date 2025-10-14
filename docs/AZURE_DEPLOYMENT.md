# CodeJudge: Azure Deployment Guide

This guide explains how to deploy CodeJudge to Microsoft Azure using either Docker or a pre-built binary. It is written for third-party users and omits internal or irrelevant details.



## Prerequisites## Prerequisites

- Azure CLI installed and logged in (`az login`)- Azure CLI installed and logged in (`az login`)

- Docker installed and running (for Docker-based deployment)- Docker installed and running (for Docker-based deployment)

- Your CodeJudge project ready to deploy- Your CodeJudge project ready to deploy



## 1. Deploying with Docker## 1. Deploying with Docker



### Step 1: Build and Push Docker Image### Step 1: Build and Push Docker Image

1. Build your Docker image:1. Build your Docker image:

   ```sh   ```sh

   docker build -t <your-dockerhub-username>/codejudge:latest .   docker build -t <your-dockerhub-username>/codejudge:latest .

   ```   ```

2. Push the image to Docker Hub:2. Push the image to Docker Hub:

   ```sh   ```sh

   docker push <your-dockerhub-username>/codejudge:latest   docker push <your-dockerhub-username>/codejudge:latest

   ```   ```



### Step 2: Create Azure Web App for Containers### Step 2: Create Azure Web App for Containers

1. Set variables:1. Set variables:

   ```sh   ```sh

   RESOURCE_GROUP="codejudge-rg"   RESOURCE_GROUP="codejudge-rg"

   APP_NAME="codejudge-$(openssl rand -hex 4)"   APP_NAME="codejudge-$(openssl rand -hex 4)"

   PLAN="codejudge-plan"   PLAN="codejudge-plan"

   ```   ```

2. Create a resource group and app service plan (if not already created):2. Create a resource group and app service plan (if not already created):

   ```sh   ```sh

   az group create --name $RESOURCE_GROUP --location "Southeast Asia"   az group create --name $RESOURCE_GROUP --location "Southeast Asia"

   az appservice plan create --name $PLAN --resource-group $RESOURCE_GROUP --sku B1 --is-linux   az appservice plan create --name $PLAN --resource-group $RESOURCE_GROUP --sku B1 --is-linux

   ```   ```

3. Create the web app:3. Create the web app:

   ```sh   ```sh

   az webapp create --resource-group $RESOURCE_GROUP --plan $PLAN --name $APP_NAME --deployment-container-image-name <your-dockerhub-username>/codejudge:latest   az webapp create --resource-group $RESOURCE_GROUP --plan $PLAN --name $APP_NAME --deployment-container-image-name <your-dockerhub-username>/codejudge:latest

   ```   ```



### Step 3: Configure Environment Variables### Step 3: Configure Environment Variables

   ```sh   ```sh

   az webapp config appsettings set --resource-group $RESOURCE_GROUP --name $APP_NAME --settings JWT_SECRET=<your-secret> PORT=8080 WEBSITES_PORT=8080 GIN_MODE=release   az webapp config appsettings set --resource-group $RESOURCE_GROUP --name $APP_NAME --settings JWT_SECRET=<your-secret> PORT=8080 WEBSITES_PORT=8080 GIN_MODE=release

   ```   ```



### Step 4: Enable Logging (Optional)### Step 4: Enable Logging (Optional)

   ```sh   ```sh

   az webapp log config --resource-group $RESOURCE_GROUP --name $APP_NAME --application-logging true --level information   az webapp log config --resource-group $RESOURCE_GROUP --name $APP_NAME --application-logging true --level information

   az webapp log tail --resource-group $RESOURCE_GROUP --name $APP_NAME   az webapp log tail --resource-group $RESOURCE_GROUP --name $APP_NAME

   ```   ```



### Step 5: Test Your Deployment### Step 5: Test Your Deployment

   ```sh   ```sh

   az webapp browse --resource-group $RESOURCE_GROUP --name $APP_NAME   az webapp browse --resource-group $RESOURCE_GROUP --name $APP_NAME

   # Or visit: https://$APP_NAME.azurewebsites.net   # Or visit: https://$APP_NAME.azurewebsites.net

   ```   ```



## 2. Deploying with Pre-built Binary (ZIP Deploy)## 2. Deploying with Pre-built Binary (ZIP Deploy)



1. Build your Go application (example for monolith):1. Build your Go application (example for monolith):

   ```sh   ```sh

   cd services/go/monolith   cd services/go/monolith

   go build -o codejudge-monolith .   go build -o codejudge-monolith .

   ```   ```

2. Create a startup script:2. Create a startup script:

   ```sh   ```sh

   echo '#!/bin/sh\n./codejudge-monolith' > startup.sh   echo '#!/bin/sh\n./codejudge-monolith' > startup.sh

   chmod +x startup.sh   chmod +x startup.sh

   ```   ```

3. Create a ZIP archive:3. Create a ZIP archive:

   ```sh   ```sh

   zip deploy.zip codejudge-monolith startup.sh static/*   zip deploy.zip codejudge-monolith startup.sh static/*

   ```   ```

4. Deploy ZIP to Azure:4. Deploy ZIP to Azure:

   ```sh   ```sh

   az webapp deploy --resource-group $RESOURCE_GROUP --name $APP_NAME --src-path deploy.zip --type zip   az webapp deploy --resource-group $RESOURCE_GROUP --name $APP_NAME --src-path deploy.zip --type zip

   ```   ```



## 3. Cost Management## 3. Cost Management

- B1 App Service Plan: ~$13/month (as of 2025)- B1 App Service Plan: ~$13/month (as of 2025)

- You can stop the app service plan to save costs:- You can stop the app service plan to save costs:

   ```sh   ```sh

   az appservice plan update --name $PLAN --resource-group $RESOURCE_GROUP --sku FREE   az appservice plan update --name $PLAN --resource-group $RESOURCE_GROUP --sku FREE

   ```   ```

- Or delete the plan when not needed:- Or delete the plan when not needed:

   ```sh   ```sh

   az appservice plan delete --name $PLAN --resource-group $RESOURCE_GROUP   az appservice plan delete --name $PLAN --resource-group $RESOURCE_GROUP

   ```   ```



## 4. Troubleshooting## 4. Troubleshooting

- View logs:- View logs:

   ```sh   ```sh

   az webapp log tail --resource-group $RESOURCE_GROUP --name $APP_NAME   az webapp log tail --resource-group $RESOURCE_GROUP --name $APP_NAME

   ```   ```

- Restart the app:- Restart the app:

   ```sh   ```sh

   az webapp restart --resource-group $RESOURCE_GROUP --name $APP_NAME   az webapp restart --resource-group $RESOURCE_GROUP --name $APP_NAME

   ```   ```

- Delete the app:- Delete the app:

   ```sh   ```sh

   az webapp delete --resource-group $RESOURCE_GROUP --name $APP_NAME   az webapp delete --resource-group $RESOURCE_GROUP --name $APP_NAME

   ```   ```

- Delete the resource group:- Delete the resource group:

   ```sh   ```sh

   az group delete --name $RESOURCE_GROUP --yes --no-wait   az group delete --name $RESOURCE_GROUP --yes --no-wait

   ```   ```



## Notes## Notes

- You can create multiple web apps on the same plan at no extra cost.- You can create multiple web apps on the same plan at no extra cost.

- Free tier (F1) is available but has limitations.- Free tier (F1) is available but has limitations.

- B1 tier is recommended for better performance and cost-effectiveness.- B1 tier is recommended for better performance and cost-effectiveness.



For more details, see the official Azure documentation: https://docs.microsoft.com/azure/app-service/containers/For more details, see the official Azure documentation: https://docs.microsoft.com/azure/app-service/containers/