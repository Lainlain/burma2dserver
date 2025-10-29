# 🎰 Burma2D Lottery Server

A real-time lottery data server built with **Go** and **Gin framework** that streams lottery updates using **Server-Sent Events (SSE)**.

---

## 📋 Project Summary

### ✅ What Was Built

1. **Go Project Structure**
   - Created complete Go project in `/Go` folder
   - Implemented `live` package for lottery management
   - Used Gin web framework for HTTP routing

2. **Core Features**
   - ✅ RESTful API endpoints for lottery data
   - ✅ Server-Sent Events (SSE) for real-time streaming
   - ✅ POST endpoint to receive lottery updates
   - ✅ GET endpoint to retrieve current lottery data
   - ✅ Health check endpoint

3. **Live Package (`/Go/live/`)**
   - `LotteryData` struct with JSON serialization
   - SSE stream manager with concurrent client handling
   - Thread-safe data updates with mutex locks
   - Real-time broadcasting to all connected clients

---

## 📁 Project Structure

```
Go/
├── main.go                 # Main server entry point
├── go.mod                  # Go module dependencies
├── go.sum                  # Dependency checksums
├── test-api.sh            # API testing script
├── burma2d-server    # Compiled binary
└── live/
    └── lottery.go         # Live lottery package (SSE + data management)
```

---

## 🚀 API Endpoints

### 1. Health Check
```bash
GET /
Response: {"message": "Burma2D Lottery Server", "status": "running"}
```

### 2. Get Current Live Data
```bash
GET /api/burma2d/live
Response: Returns current lottery data in JSON format
```

### 3. Update Data (POST - Internal)
```bash
POST /api/burma2d/update
Content-Type: application/json

Body:
{
  "live": "22",
  "status": "On",
  "set1200": "15",
  "value1200": "89",
  "result1200": "589",
  "set430": "67",
  "value430": "34",
  "result430": "134",
  "modern930": "845",
  "internet930": "921",
  "modern200": "376",
  "internet200": "542",
  "updatetime": "12:01:45 16/10/2025"
}
```

### 4. Real-Time SSE Stream 📡
```bash
GET /api/burma2d/stream
Content-Type: text/event-stream

# Streams real-time lottery updates to all connected clients
# Each update is sent as an SSE event
```

---

## 🛠️ Technical Implementation

### SSE Stream Manager
- **Concurrent client handling** using goroutines
- **Channel-based broadcasting** for efficient updates
- **Automatic client cleanup** on disconnect
- **Thread-safe operations** with mutex locks

### Data Model
```go
type LotteryData struct {
    Live        string `json:"live"`
    Status      string `json:"status"`
    Set1200     string `json:"set1200"`
    Value1200   string `json:"value1200"`
    Result1200  string `json:"result1200"`
    Set430      string `json:"set430"`
    Value430    string `json:"value430"`
    Result430   string `json:"result430"`
    Modern930   string `json:"modern930"`
    Internet930 string `json:"internet930"`
    Modern200   string `json:"modern200"`
    Internet200 string `json:"internet200"`
    UpdateTime  string `json:"updatetime"`
}
```

---

## 🏃 How to Run

### 1. Build the Server
```bash
cd "/home/lainlain/Desktop/Go Lang /Burma2D/Go"
go build -o burma2d-server
```

### 2. Run the Server
```bash
./burma2d-server
```
Server starts on `http://localhost:4545`

### 3. Test the API
```bash
chmod +x test-api.sh
./test-api.sh
```

### 4. Test SSE Streaming (Open separate terminal)
```bash
curl -N http://localhost:4545/api/burma2d/stream
```

### 5. Send Lottery Updates
```bash
curl -X POST http://localhost:4545/api/burma2d/update \
  -H "Content-Type: application/json" \
  -d '{
    "live": "22",
    "status": "On",
    "set1200": "15",
    "value1200": "89",
    "result1200": "589",
    "set430": "67",
    "value430": "34",
    "result430": "134",
    "modern930": "845",
    "internet930": "921",
    "modern200": "376",
    "internet200": "542",
    "updatetime": "12:01:45 16/10/2025"
  }'
```

---

## 🔄 How SSE Streaming Works

1. **Client connects** to `/api/burma2d/stream`
2. **Client added to active clients map** with mutex protection
3. **When data updates** via POST to `/api/burma2d/update`:

---

## 🎯 Usage Example

**For client apps (Vue/Kotlin):**
- Connect clients to `http://localhost:4545/api/burma2d/stream`
- Auto-receive updates when server broadcasts

**For API data source:**
- POST lottery results to `/api/burma2d/update`

---

## 📦 Dependencies

```
github.com/gin-gonic/gin v1.10.0
```

Automatically installed when building with `go build`

---

## 🎉 Completed Features

✅ Go project with Gin framework  
✅ Live package for lottery management  
✅ POST route to receive apiexample data  
✅ SSE streaming for real-time updates  
✅ Thread-safe concurrent client handling  
✅ Health check and current data endpoints  
✅ Complete test script included  

---

## 📝 Next Steps (Optional)

- Add authentication/authorization
- Implement database persistence (PostgreSQL/MongoDB)
- Add rate limiting for POST endpoint
- Create admin dashboard UI
- Add logging middleware
- Deploy to production server

---

## 👨‍💻 Development Info

- **Framework**: Gin (Go web framework)
- **Language**: Go 1.x
- **Architecture**: RESTful API with SSE
- **Concurrency**: Goroutines + Channels
- **Port**: 8080

---

## 🧪 Testing Workflow

1. **Open terminal 1**: `go run main.go` (server running)
2. **Open terminal 2**: `curl -N http://localhost:4545/api/burma2d/stream` (keep open)

---

**Server is ready to stream lottery data in real-time! 🚀**
# thaimasterserver
# burma2dserver
