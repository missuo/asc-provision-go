# asc-provision-go

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/missuo/asc-provision-go)](https://goreportcard.com/report/github.com/missuo/asc-provision-go)
[![GoDoc](https://godoc.org/github.com/missuo/asc-provision-go?status.svg)](https://godoc.org/github.com/missuo/asc-provision-go)

A Go API server for managing Apple Developer Account devices and provisioning profiles.

## TODO

- [ ] Add dashboard for managing devices and profiles
- [ ] Integrate with [resign](https://github.com/missuo/resign)

## Integration

I have already integrated this API into my iOS app, you can see the preview below.

![Preview](./Screenshots/Preview.JPEG)

Open source at [https://github.com/missuo/Devices](https://github.com/missuo/Devices)

## Features

- 📱 **Device Management**
  - List all iOS/macOS devices
  - Register new devices
  - Filter devices by platform
  
- 📃 **Provisioning Profile Management**
  - List all profiles
  - Create new profiles with all compatible devices
  - Download profiles as .mobileprovision files
  - Delete profiles
  
- 🔐 **Authentication**
  - JWT token generation for Apple API authentication
  - Automatic token renewal

- 🌐 **RESTful API**
  - Simple and easy-to-use HTTP endpoints
  - JSON response format

## Prerequisites

- Go 1.23 or higher
- Apple Developer Account
- App Store Connect API Key (with Admin role)

## Installation

```bash
# Clone the repository
git clone https://github.com/missuo/asc-provision-go.git
cd asc-provision-go

# Install dependencies
go mod tidy

# Build the application
go build -o asc-provision-go
```

## Configuration

Create a `.env` file in the project root with the following variables:

```
APPLE_KEY_ID=YOUR_KEY_ID
APPLE_ISSUER_ID=YOUR_TEAM_ID
APPLE_PRIVATE_KEY_PATH=./path/to/AuthKey_XXX.p8
APPLE_BUNDLE_ID=com.your.bundleid
PORT=8080
```

| Variable | Description |
|----------|-------------|
| `APPLE_KEY_ID` | Key ID from App Store Connect |
| `APPLE_ISSUER_ID` | Your Team ID (10-character identifier) |
| `APPLE_PRIVATE_KEY_PATH` | Path to your .p8 private key file |
| `APPLE_BUNDLE_ID` | Bundle ID to use when creating profiles |
| `API_KEY` | API key for authentication |
| `PORT` | (Optional) Server port number, defaults to 8080 |

## Usage

### Running the Server

```bash
./asc-provision-go
```

The server will start on the configured port (default: 8080).

### API Endpoints

#### Device Management

- **List all devices**
  ```
  GET /api/devices?platform=IOS
  ```
  Query parameters:
  - `platform` (optional): Filter by platform, values can be `IOS` or `MAC_OS`

- **Register a new device**
  ```
  POST /api/devices
  ```
  Request body:
  ```json
  {
    "name": "Device Name",
    "udid": "Device UDID",
    "platform": "IOS"
  }
  ```

#### Profile Management

- **List all provisioning profiles**
  ```
  GET /api/profiles
  ```

- **Get a specific profile**
  ```
  GET /api/profiles/:id
  ```

- **Create a new profile**
  ```
  POST /api/profiles
  ```
  Request body:
  ```json
  {
    "name": "Profile Name",
    "profileType": "IOS_APP_ADHOC"
  }
  ```
  Supported profile types:
  - `IOS_APP_ADHOC`
  - `IOS_APP_DEVELOPMENT`
  - `IOS_APP_STORE`
  - `IOS_APP_INHOUSE` (Enterprise accounts only)
  - `MAC_APP_DEVELOPMENT`
  - `MAC_APP_STORE`
  - `MAC_APP_DIRECT`

- **Delete a profile**
  ```
  DELETE /api/profiles/:id
  ```

- **Download a profile**
  ```
  GET /api/profiles/:id/download
  ```

- **Create and download a profile in one step**
  ```
  POST /api/profiles/create-and-download
  ```

## Example Usage with Curl

### List iOS Devices

```bash
curl -X GET "http://localhost:8080/api/devices?platform=IOS" \
  -H "Content-Type: application/json"
```

### Register a New Device

```bash
curl -X POST "http://localhost:8080/api/devices" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "iPhone 15 Pro",
    "udid": "00000000-1111111111111111",
    "platform": "IOS"
  }'
```

### Create a New Ad Hoc Profile

```bash
curl -X POST "http://localhost:8080/api/profiles" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Ad Hoc Profile",
    "profileType": "IOS_APP_ADHOC"
  }'
```

### Download a Profile

```bash
curl -X GET "http://localhost:8080/api/profiles/PROFILE_ID/download" \
  -H "Content-Type: application/json" \
  -o "profile.mobileprovision"
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Apple App Store Connect API](https://developer.apple.com/documentation/appstoreconnectapi)
- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [JWT Go](https://github.com/golang-jwt/jwt)