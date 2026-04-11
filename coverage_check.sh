#!/bin/bash

echo "========================================"
echo "CODE COVERAGE SUMMARY"
echo "========================================"
echo ""

# MQ Coverage
echo "MQ Coverage:"
cd /Users/mayur/Documents/Projects/src/gpu-pipeline/mq
go test ./internal -coverprofile=coverage.out 2>&1 > /dev/null
go tool cover -func=coverage.out | grep "total:" | awk '{print "  " $0}'

# Collector Coverage
echo ""
echo "Collector Coverage:"
cd /Users/mayur/Documents/Projects/src/gpu-pipeline/collector
go test ./internal -timeout 30s -coverprofile=coverage.out 2>&1 > /dev/null
go tool cover -func=coverage.out | grep "total:" | awk '{print "  " $0}'

# Streamer Coverage
echo ""
echo "Streamer Coverage:"
cd /Users/mayur/Documents/Projects/src/gpu-pipeline/streamer
go test ./internal -coverprofile=coverage.out 2>&1 > /dev/null
go tool cover -func=coverage.out | grep "total:" | awk '{print "  " $0}'

echo ""
echo "========================================"
echo "TARGET: 80% for all services"
echo "========================================"
