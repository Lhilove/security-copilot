#!/bin/bash
echo "Generating ENCRYPTION_KEY and JWT_SECRET..."
echo ""
echo "ENCRYPTION_KEY=$(openssl rand -base64 32)"
echo "JWT_SECRET=$(openssl rand -base64 32)"
echo ""
echo "Copy these into your .env file."