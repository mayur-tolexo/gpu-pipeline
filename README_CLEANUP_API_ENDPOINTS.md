# README.md Updates - Remove Duplicates & Update API Endpoints

**Date**: April 12, 2026  
**Status**: ✅ Complete  

---

## Changes Made

### 1. ✅ Removed Duplicate "Access Swagger UI" Section

**Before**: Two identical "Access Swagger UI" sections at lines 618 and 744
**After**: Single "Access Swagger UI" section at line 618

**Removed Content**:
- Redundant second section with duplicate port-forward instructions
- Duplicate example requests
- Second "Deployment" section that followed it

**Result**: README cleaned up, no duplicate information

---

### 2. ✅ Removed POST Endpoints (No Longer Supported)

The API Gateway has removed POST endpoints. Updated all references:

#### Before (Incorrect)
```bash
# This endpoint was removed from api-gateway
curl -X POST http://localhost:8000/api/v1/telemetry/query \
  -H "Content-Type: application/json" \
  -d '{
    "gpu_id": "gpu-001",
    "start_time": "2026-04-11T00:00:00Z",
    "end_time": "2026-04-12T00:00:00Z"
  }'
```

#### After (Correct - GET Only)
```bash
# All endpoints are now GET-only
curl -X GET "http://localhost:8000/api/v1/gpus/gpu-001/telemetry?start_time=2026-04-11T00:00:00Z&end_time=2026-04-12T00:00:00Z"
```

---

### 3. ✅ Updated API Endpoints List

#### Before
```markdown
### API Endpoints
- **Swagger UI**: `http://localhost:8000/swagger/`
- **Health Check**: `http://localhost:8000/api/v1/health`
- **List GPUs**: `GET http://localhost:8000/api/v1/gpus`
- **Query Telemetry**: `POST http://localhost:8000/api/v1/telemetry/query`
```

#### After
```markdown
### API Endpoints (GET Only)
- **Swagger UI**: `http://localhost:8000/swagger/`
- **Health Check**: `GET /api/v1/health`
- **List GPUs**: `GET /api/v1/gpus`
- **Get GPU Telemetry**: `GET /api/v1/gpus/{gpu_id}/telemetry?start_time=...&end_time=...`
```

---

### 4. ✅ Fixed Swagger UI URL

#### Before (Incorrect)
```bash
# Or manually visit: http://localhost:8081/swagger/
```

#### After (Correct)
```bash
# Or manually visit: http://localhost:8000/swagger/
```

---

### 5. ✅ Simplified Example Requests

Removed unnecessary `-H "Content-Type: application/json"` headers from GET requests and POST request that no longer exists.

#### Before
```bash
curl -X GET http://localhost:8000/api/v1/gpus \
  -H "Content-Type: application/json"

curl -X GET "http://localhost:8000/api/v1/gpus/gpu-001/telemetry?..." \
  -H "Content-Type: application/json"

curl -X POST http://localhost:8000/api/v1/telemetry/query \
  -H "Content-Type: application/json" \
  -d '{...}'
```

#### After
```bash
curl -X GET http://localhost:8000/api/v1/gpus

curl -X GET "http://localhost:8000/api/v1/gpus/gpu-001/telemetry?..."

# POST endpoint removed - no longer supported
```

---

## Files Modified

| File | Changes |
|------|---------|
| `README.md` | ✅ Removed duplicate "Access Swagger UI" section<br>✅ Removed POST endpoint references<br>✅ Updated to GET-only endpoints<br>✅ Fixed Swagger UI URL<br>✅ Simplified example requests |

---

## Summary of Duplicate Removals

| Item | Before | After | Status |
|------|--------|-------|--------|
| "Access Swagger UI" sections | 2 (duplicate) | 1 | ✅ Cleaned |
| "Build & Test" sections | 2 (duplicate) | 1 | ✅ Cleaned (previous update) |
| "Services" sections | 2 (duplicate) | 1 | ✅ Cleaned (previous update) |
| POST endpoints documented | Yes | No | ✅ Removed |
| Swagger URL accuracy | http://8081 (wrong) | http://8000 (correct) | ✅ Fixed |

---

## Current API Endpoints (All GET)

```
GET  /api/v1/health                                    # Health check
GET  /api/v1/gpus                                      # List all GPU IDs
GET  /api/v1/gpus/{gpu_id}/telemetry?...              # Get GPU telemetry with time filters
```

---

## Next Steps

1. Update service READMEs to reflect GET-only API
2. Update API Gateway implementation to align with documentation
3. Consider adding POST endpoints in future if needed
4. Update client SDKs to use GET-based query endpoints

---

**Status**: ✅ **COMPLETE - DUPLICATES REMOVED, API ENDPOINTS UPDATED**
